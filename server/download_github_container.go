package server

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"

	"github.com/golang-jwt/jwt/v5"
	crauthn "github.com/google/go-containerregistry/pkg/authn"
	crname "github.com/google/go-containerregistry/pkg/name"
	cr "github.com/google/go-containerregistry/pkg/v1"
	crremote "github.com/google/go-containerregistry/pkg/v1/remote"
	crtarball "github.com/google/go-containerregistry/pkg/v1/tarball"
	"go.uber.org/zap"
)

func (s *Server) downloadGithubContainer(
	ctx context.Context,
	j *job.SyncContainerRegistryPackage,
) (string, error) {
	l := logutils.LoggerFromContext(ctx)

	var downloadsDir string
	{ // create container downloads dir
		downloadsDir = strings.ReplaceAll(filepath.Join(
			s.cfg.Dir.Downloads, j.GetRepoOwner(), j.GetRepo(), "containers", j.GetPackageName(), j.GetDigest(),
		), ":", "-")
		if err := os.MkdirAll(downloadsDir, 0750); err != nil {
			return "", fmt.Errorf("failed to create container download directory: %s: %w",
				downloadsDir, err,
			)
		}
	}

	var auth crauthn.Authenticator
	{ // get token
		_jwt, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iat": time.Now().Unix(),
			"exp": time.Now().Add(10 * time.Minute).Unix(),
			"iss": s.cfg.Github.App.ID,
		}).SignedString(s.cfg.Github.App.RsaPrivateKey())
		if err != nil {
			return "", fmt.Errorf("failed to sign a jwt: %w", err)
		}

		token, res, err := s.github.WithAuthToken(_jwt).Apps.CreateInstallationToken(
			ctx, s.cfg.Github.App.InstallationID, nil,
		)
		if err == nil && res.StatusCode != http.StatusCreated {
			err = fmt.Errorf("unexpected http status: %d", res.StatusCode)
		}
		if err != nil {
			return "", fmt.Errorf("failed to get auth token: %w", err)
		}

		auth = crauthn.FromConfig(crauthn.AuthConfig{
			Username: "oauth2accesstoken",
			Password: token.GetToken(),
		})
	}

	var ref crname.Reference
	{ // get reference
		_ref, err := crname.ParseReference(j.GetPackageUrl())
		if err != nil {
			return "", fmt.Errorf("failed to parse container image url: %s: %w",
				j.GetPackageUrl(), err,
			)
		}
		ref = _ref
	}

	var desc *crremote.Descriptor
	{ // get descriptor
		_desc, err := crremote.Get(ref, crremote.WithAuth(auth))
		if err != nil {
			return "", fmt.Errorf("failed to get a descriptor for container image: %s: %w",
				j.GetPackageUrl(), err,
			)
		}
		desc = _desc
	}

	var indexManifest *cr.IndexManifest
	var images = make(map[string][]cr.Image)
	{ // get images
		switch {
		case desc.MediaType.IsImage():
			image, err := crremote.Image(ref, crremote.WithAuth(auth))
			if err != nil {
				return "", fmt.Errorf("failed to retrieve container image: %s: %w",
					j.GetPackageUrl(), err,
				)
			}

			platform := "unknown/unknown"
			if desc.Platform != nil {
				platform = desc.Platform.String()
			}
			if _, exists := images[platform]; !exists {
				images[platform] = make([]cr.Image, 0)
			}

			images[platform] = append(images[platform], image)

			l.Debug("Downloaded a manifest",
				zap.String("digest", desc.Digest.String()),
				zap.String("platform", platform),
				zap.Any("annotations", desc.Annotations),
			)

		case desc.MediaType.IsIndex():
			index, err := crremote.Index(ref, crremote.WithAuth(auth))
			if err != nil {
				return "", fmt.Errorf("failed to retrieve container index: %s: %w",
					j.GetPackageUrl(), err,
				)
			}

			indexManifest, err = index.IndexManifest()
			if err != nil {
				return "", fmt.Errorf("failed to get image index manifest from a descriptor: %s: %s: %w",
					j.GetPackageUrl(), desc.Digest.String(), err,
				)
			}

			l.Debug("Downloaded an index",
				zap.String("digest", desc.Digest.String()),
				zap.String("reference", ref.Name()),
				zap.Any("annotations", indexManifest.Annotations),
			)

			for _, desc := range indexManifest.Manifests {
				if !desc.MediaType.IsImage() {
					continue
				}

				image, err := index.Image(desc.Digest)
				if err != nil {
					return "", fmt.Errorf("failed to get image from an index: %s: %s: %w",
						j.GetPackageUrl(), desc.Digest, err,
					)
				}

				platform := "unknown/unknown"
				if desc.Platform != nil {
					platform = desc.Platform.String()
				}
				if _, exists := images[platform]; !exists {
					images[platform] = make([]cr.Image, 0)
				}

				images[platform] = append(images[platform], image)

				l.Debug("Downloaded a manifest",
					zap.String("digest", desc.Digest.String()),
					zap.String("platform", platform),
					zap.Any("annotations", desc.Annotations),
				)
			}
		}
	}

	if len(images) == 0 {
		return "", fmt.Errorf("no images to download: %s",
			j.GetPackageUrl(),
		)
	}

	var fname string
	{ // download
		fname = filepath.Join(downloadsDir, desc.Digest.Hex+".zip")
		file, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			if file != nil {
				err = errors.Join(err,
					file.Close(),
				)
			}
			return "", fmt.Errorf("failed to create download file: %w", err)
		}
		defer file.Close()

		zipper := zip.NewWriter(file)
		defer zipper.Close()

		if desc.MediaType.IsIndex() {
			stream, err := zipper.Create(filepath.Join(desc.Digest.Hex + ".json"))
			if err != nil {
				return "", fmt.Errorf("failed to add index json to file: %w", err)
			}

			l.Debug("Archiving index manifest...",
				zap.String("digest", desc.Digest.String()),
			)

			if err := json.NewEncoder(stream).Encode(indexManifest); err != nil {
				return "", fmt.Errorf("failed to write index json to file: %w", err)
			}
		}

		for platform, _images := range images {
			if platform == "" {
				continue
			}

			for _, image := range _images {
				digest, err := image.Digest()
				if err != nil {
					return "", fmt.Errorf("failed to get image digest: %s: %w",
						j.GetPackageUrl(), err,
					)
				}

				stream, err := zipper.Create(
					filepath.Join(platform, strings.ReplaceAll(digest.String(), ":", "-")+".tar"),
				)
				if err != nil {
					return "", fmt.Errorf("failed to add container tarball to file: %w", err)
				}

				l.Debug("Archiving container manifest...",
					zap.String("digest", digest.String()),
				)

				start := time.Now()

				if err := crtarball.Write(ref, image, stream); err != nil {
					return "", fmt.Errorf("failed to write container tarball: %w", err)
				}

				l.Debug("Archived a container manifest",
					zap.String("digest", digest.String()),
					zap.Duration("duration", time.Since(start)),
				)
			}
		}
	}

	return fname, nil
}
