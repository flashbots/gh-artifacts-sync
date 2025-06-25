package server

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
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

	errs := make([]error, 0)

	var images = make(map[string]cr.Image, 0)
	var configs = make(map[string]*cr.ConfigFile, 0)
	var digests = make(map[string]cr.Hash, 0)
	{ // get images
		for _, f := range z.File {
			if f.FileInfo().IsDir() {
				continue
			}

			platform := filepath.Dir(f.Name)

			if len(dst.Platforms) > 0 && !dst.HasPlatform(platform) {
				l.Info("Ignoring platform that is not mentioned in the list",
					zap.String("platform", platform),
				)
				continue
			}

			image, err := crtarball.Image(zipFileOpener(f), nil)
			if err != nil {
				l.Error("Failed to open container tarball",
					zap.Error(err),
					zap.String("file", f.Name),
				)
				errs = append(errs, err)
				continue
			}

			config, err := image.ConfigFile()
			if err != nil {
				l.Error("Failed to get container's config file",
					zap.Error(err),
					zap.String("file", f.Name),
				)
				errs = append(errs, err)
				continue
			}

			digest, err := image.Digest()
			if err != nil {
				l.Error("Failed to get container's digest",
					zap.Error(err),
					zap.String("file", f.Name),
				)
				errs = append(errs, err)
				continue
			}

			images[platform] = image
			configs[platform] = config
			digests[platform] = digest
		}
	}

	if len(images) == 0 {
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
		switch len(images) {
		case 1: // single image
			for _, image := range images {
				// 1-iteration loop b/c there's only one
				l.Debug("Pushing container image to the destination",
					zap.String("reference", ref.String()),
				)
				uploadErr = crremote.Write(ref, image, crremote.WithAuth(auth))
				if uploadErr != nil {
					l.Error("Failed to push container image to the destination",
						zap.Error(uploadErr),
						zap.String("reference", ref.String()),
					)
				}
				l.Info("Pushed container image to the destination",
					zap.String("reference", ref.String()),
				)
			}

		default: // multi-platform image
			var index cr.ImageIndex = crempty.Index
			for platform, image := range images {
				desc := cr.Descriptor{
					Platform: configs[platform].Platform(),
					Digest:   digests[platform],
				}
				index = crmutate.AppendManifests(index, crmutate.IndexAddendum{
					Add:        image,
					Descriptor: desc,
				})
			}

			l.Debug("Pushing container image index to the destination",
				zap.String("reference", ref.String()),
			)
			uploadErr = crremote.WriteIndex(ref, index, crremote.WithAuth(auth))
			if uploadErr != nil {
				l.Error("Failed to push container image index to the destination",
					zap.Error(uploadErr),
					zap.String("reference", ref.String()),
				)
			}
			l.Info("Pushed container image index to the destination",
				zap.String("reference", ref.String()),
			)
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
