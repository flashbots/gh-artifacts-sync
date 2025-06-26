package server

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"

	"go.uber.org/zap"

	crauthn "github.com/google/go-containerregistry/pkg/authn"
	crname "github.com/google/go-containerregistry/pkg/name"
	cr "github.com/google/go-containerregistry/pkg/v1"
	crempty "github.com/google/go-containerregistry/pkg/v1/empty"
	crmutate "github.com/google/go-containerregistry/pkg/v1/mutate"
	crremote "github.com/google/go-containerregistry/pkg/v1/remote"
	crtransport "github.com/google/go-containerregistry/pkg/v1/remote/transport"
	crtarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

type container struct {
	config   *cr.ConfigFile
	digest   cr.Hash
	image    cr.Image
	manifest *cr.Manifest
}

func (s *Server) uploadFromZipToGcpArtifactRegistryDocker(
	ctx context.Context,
	j job.UploadableContainer,
	zname string,
	dst *config.Destination,
) error {
	l := logutils.LoggerFromContext(ctx)

	var z *zip.ReadCloser
	{ // open archive
		_z, err := zip.OpenReader(zname)
		if err != nil {
			return fmt.Errorf("failed to open zip file: %w", err)
		}
		defer _z.Close()
		z = _z
	}

	var (
		attestations = make(map[string][]*container, 0)
		containers   = make(map[string]*container, 0)
		errs         = make([]error, 0)
	)
	{ // get images
		for _, f := range z.File {
			if f.FileInfo().IsDir() {
				continue
			}

			l := l.With(
				zap.String("file", f.Name),
			)

			image, err := crtarball.Image(zipFileOpener(f), nil)
			if err != nil {
				l.Error("Failed to open container tarball", zap.Error(err))
				errs = append(errs, err)
				continue
			}

			manifest, err := image.Manifest()
			if err != nil {
				l.Error("Failed to get container's manifest", zap.Error(err))
				errs = append(errs, err)
				continue
			}

			config, err := image.ConfigFile()
			if err != nil {
				l.Error("Failed to get container's config file", zap.Error(err))
				errs = append(errs, err)
				continue
			}

			digest, err := image.Digest()
			if err != nil {
				l.Error("Failed to get container's digest", zap.Error(err))
				errs = append(errs, err)
				continue
			}

			c := &container{
				config:   config,
				digest:   digest,
				image:    image,
				manifest: manifest,
			}

			if len(dst.Platforms) == 0 { // no filtering, include all
				containers[digest.String()] = c
				l.Debug("Including a manifest",
					zap.String("digest", digest.String()),
					zap.String("platform", config.Platform().String()),
					zap.Any("annotations", manifest.Annotations),
				)
				continue
			}

			if dst.HasPlatform(config.Platform()) {
				containers[digest.String()] = c
				for _, attestation := range attestations[digest.String()] {
					containers[attestation.digest.String()] = attestation
					l.Debug("Including an attestation",
						zap.String("digest", attestation.digest.String()),
						zap.Any("annotations", attestation.manifest.Annotations),
					)
				}
				delete(attestations, digest.String())
				continue
			}

			if manifest.Annotations["vnd.docker.reference.type"] == "attestation-manifest" {
				if ref := manifest.Annotations["vnd.docker.reference.digest"]; ref != "" {
					if _, alreadySeen := containers[ref]; alreadySeen {
						containers[digest.String()] = c
						l.Debug("Including an attestation",
							zap.String("digest", digest.String()),
							zap.Any("annotations", manifest.Annotations),
						)
						continue
					}

					// maybe it will come after
					if _, exists := attestations[ref]; !exists {
						attestations[ref] = make([]*container, 0)
					}
					attestations[ref] = append(attestations[ref], c)
				}
			}
		}
	}

	if len(containers) == 0 {
		l.Debug("No matching platforms, skipping...")
		return utils.FlattenErrors(errs)
	}

	var ref crname.Reference
	{ // get remote reference
		reference := j.GetDestinationReference(dst)
		_ref, err := crname.ParseReference(reference)
		if err != nil {
			l.Error("Failed to parse destination reference",
				zap.Error(err),
				zap.String("reference", reference),
			)
			errs = append(errs, err)
			return utils.FlattenErrors(errs)
		}
		ref = _ref
	}

	var auth crauthn.Authenticator
	{ // get authentication token
		token, err := utils.WithTimeout(ctx, 10*time.Minute, func(ctx context.Context) (string, error) {
			return s.gcp.AccessToken(ctx, "https://www.googleapis.com/auth/cloud-platform")
		})
		if err != nil {
			l.Error("Failed to get gcp token", zap.Error(err))
			errs = append(errs, err)
			return utils.FlattenErrors(errs)
		}
		auth = crauthn.FromConfig(crauthn.AuthConfig{
			Username: "oauth2accesstoken",
			Password: token,
		})
	}

	var uploadErr error
	{ // upload
		switch len(containers) {
		case 1: // single image
			for _, c := range containers {
				l := l.With(
					zap.String("reference", ref.String()),
					zap.String("digest", c.digest.String()),
					zap.String("platform", c.config.Platform().String()),
				)
				// 1-iteration loop b/c there's only one
				l.Debug("Pushing container manifest to the destination")
				uploadErr = crremote.Write(ref, c.image, crremote.WithAuth(auth))
				if uploadErr != nil {
					l.Error("Failed to push container image to the destination", zap.Error(uploadErr))
				}
				l.Info("Pushed container image to the destination")
			}

		default: // multi-platform image
			l := l.With(
				zap.String("reference", ref.String()),
			)
			var index cr.ImageIndex = crempty.Index
			for _, c := range containers {
				index = crmutate.AppendManifests(index, crmutate.IndexAddendum{
					Add: c.image,

					Descriptor: cr.Descriptor{
						Annotations: c.manifest.Annotations,
						Digest:      c.digest,
						Platform:    c.config.Platform(),
					},
				})
			}

			l.Debug("Pushing container index to the destination")
			uploadErr = crremote.WriteIndex(ref, index, crremote.WithAuth(auth))
			if uploadErr != nil {
				l.Error("Failed to push container image index to the destination", zap.Error(uploadErr))
			}
			l.Info("Pushed container image index to the destination")
		}
	}

	if uploadErr != nil {
		transportErr := &crtransport.Error{}
		if errors.As(uploadErr, &transportErr) && transportErr.Temporary() {
			errs = append(errs, uploadErr)
		} else {
			errs = append(errs, utils.NoRetry(uploadErr))
		}
	}

	return utils.FlattenErrors(errs)
}

func zipFileOpener(zf *zip.File) crtarball.Opener {
	return func() (io.ReadCloser, error) {
		return zf.Open()
	}
}
