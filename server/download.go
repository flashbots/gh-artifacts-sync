package server

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

func (s *Server) downloadGithubContainerRegistryPackage(
	ctx context.Context,
	j *job.SyncContainerRegistryPackage,
) (string, error) {
	l := logutils.LoggerFromContext(ctx)

	var downloadsDir string
	{ // create container downloads dir
		downloadsDir = filepath.Join(
			s.cfg.Dir.Downloads, j.GetRepoOwner(), j.GetRepo(), "containers", j.GetPackageName(), j.GetTag(),
		)
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
		_ref, err := crname.ParseReference(*j.Package.PackageVersion.PackageURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse container image url: %s: %w",
				*j.Package.PackageVersion.PackageURL, err,
			)
		}
		ref = _ref
	}

	var desc *crremote.Descriptor
	{ // get descriptor
		_desc, err := crremote.Get(ref, crremote.WithAuth(auth))
		if err != nil {
			return "", fmt.Errorf("failed to get a descriptor for container image: %s: %w",
				*j.Package.PackageVersion.PackageURL, err,
			)
		}
		desc = _desc
	}

	var images = make(map[string][]cr.Image)
	{ // get images
		switch {
		case desc.MediaType.IsImage():
			image, err := crremote.Image(ref, crremote.WithAuth(auth))
			if err != nil {
				return "", fmt.Errorf("failed to retrieve container image: %w", err)
			}
			platform := "unknown/unknown"
			if desc.Platform != nil {
				platform = desc.Platform.String()
			}
			if _, exists := images[platform]; !exists {
				images[platform] = make([]cr.Image, 0)
			}
			images[platform] = append(images[platform], image)

		case desc.MediaType.IsIndex():
			index, err := crremote.Index(ref, crremote.WithAuth(auth))
			if err != nil {
				return "", fmt.Errorf("failed to retrieve container index: %w", err)
			}

			indexManifest, err := index.IndexManifest()
			if err != nil {
				return "", fmt.Errorf("failed to get image index manifest from a descriptor: %s: %s: %w",
					*j.Package.PackageVersion.PackageURL, desc.Digest.String(), err,
				)
			}

			for _, desc := range indexManifest.Manifests {
				if !desc.MediaType.IsImage() {
					continue
				}
				image, err := index.Image(desc.Digest)
				if err != nil {
					return "", fmt.Errorf("failed to get image from an index: %s: %s: %w",
						*j.Package.PackageVersion.PackageURL, desc.Digest, err,
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
			}
		}
	}

	if len(images) == 0 {
		return "", fmt.Errorf("no images to download: %s",
			*j.Package.PackageVersion.PackageURL,
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

		for platform, _images := range images {
			for _, image := range _images {
				digest, err := image.Digest()
				if err != nil {
					return "", fmt.Errorf("failed to get image digest: %s: %w",
						*j.Package.PackageVersion.PackageURL, err,
					)
				}

				stream, err := zipper.Create(filepath.Join(platform, digest.Hex+".tar"))
				if err != nil {
					return "", fmt.Errorf("failed to create add container tarball to file: %w", err)
				}

				l.Debug("Downloading container image...",
					zap.String("digest", digest.String()),
				)

				start := time.Now()

				if err := crtarball.Write(ref, image, stream); err != nil {
					return "", fmt.Errorf("failed to write container tarball: %w", err)
				}

				l.Info("Downloaded a container image",
					zap.String("digest", digest.String()),
					zap.Duration("duration", time.Since(start)),
				)
			}
		}
	}

	return fname, nil
}

func (s *Server) downloadGithubReleaseAsset(
	ctx context.Context,
	j *job.SyncReleaseAsset,
) (string, error) {
	var downloadsDir string
	{ // create asset downloads dir
		downloadsDir = filepath.Join(
			s.cfg.Dir.Downloads, j.GetRepoOwner(), j.GetRepo(), "assets", strconv.Itoa(int(j.GetAssetID())),
		)
		if err := os.MkdirAll(downloadsDir, 0750); err != nil {
			return "", fmt.Errorf("failed to create asset download directory: %s: %w",
				downloadsDir, err,
			)
		}
	}

	var fname string
	{ // download
		fname = filepath.Join(downloadsDir, j.GetAssetName())
		if err := s.downloadGithubUrl(ctx, *j.Asset.URL, fname, time.Minute); err != nil {
			return "", fmt.Errorf("failed to download an asset: %w", err)
		}
	}

	return fname, nil
}

func (s *Server) downloadGithubUrl(
	ctx context.Context,
	url, fname string,
	timeout time.Duration,
) error {
	l := logutils.LoggerFromContext(ctx).With(
		zap.String("url", url),
		zap.String("file_name", fname),
	)

	var stream io.ReadCloser
	{ // get http stream
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create http request: %w", err)
		}
		req.Header.Add("accept", "application/octet-stream")
		res, err := s.github.Client().Do(req)
		if err == nil && res.StatusCode != http.StatusOK {
			err = fmt.Errorf("unexpected http status: %d", res.StatusCode)
		}
		if err != nil {
			return fmt.Errorf("failed to execute http request: %w", err)
		}
		if res != nil {
			defer res.Body.Close()
		}

		stream = res.Body
	}

	{ // download
		file, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			if file != nil {
				err = errors.Join(err,
					file.Close(),
				)
			}
			return fmt.Errorf("failed to create download file: %w", err)
		}
		defer file.Close()

		l.Debug("Downloading a file...")
		start := time.Now()

		if _, err = io.Copy(file, stream); err != nil {
			return fmt.Errorf("failed to download a file: %w", err)
		}

		l.Info("Downloaded a file",
			zap.Duration("duration", time.Since(start)),
		)
	}

	return nil
}

func (s *Server) downloadGithubWorkflowArtifact(
	ctx context.Context,
	j *job.SyncWorkflowArtifact,
) (string, error) {
	var downloadsDir string
	{ // create artifact downloads dir
		downloadsDir = filepath.Join(
			s.cfg.Dir.Downloads, j.GetRepoFullName(), "workflows", strconv.Itoa(int(j.GetWorkflowRunID())),
		)

		if err := os.MkdirAll(downloadsDir, 0750); err != nil {
			return "", fmt.Errorf("failed to create artifact download directory: %s: %w",
				downloadsDir, err,
			)
		}
	}

	var downloadLink string
	{ // get the download link
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_url, _, err := s.github.Actions.DownloadArtifact(
			ctx, j.GetRepoOwner(), j.GetRepoFullName(), j.GetArtifactID(), 16,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get the download link: %w", err)
		}
		downloadLink = _url.String()
	}

	var fname string
	{ // download
		fname = filepath.Join(downloadsDir, j.GetArtifactName())
		if err := s.downloadGithubUrl(ctx, downloadLink, fname, time.Minute); err != nil {
			return "", fmt.Errorf("failed to download an artifact: %w", err)
		}
	}

	return fname, nil
}
