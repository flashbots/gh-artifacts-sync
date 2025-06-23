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
	"time"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"

	"go.uber.org/zap"

	"google.golang.org/api/artifactregistry/v1"
	"google.golang.org/api/googleapi"

	crauthn "github.com/google/go-containerregistry/pkg/authn"
	crname "github.com/google/go-containerregistry/pkg/name"
	cr "github.com/google/go-containerregistry/pkg/v1"
	crempty "github.com/google/go-containerregistry/pkg/v1/empty"
	crmutate "github.com/google/go-containerregistry/pkg/v1/mutate"
	crremote "github.com/google/go-containerregistry/pkg/v1/remote"
	crtransport "github.com/google/go-containerregistry/pkg/v1/remote/transport"
	crtarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

func (s *Server) uploadFromZipAndDelete(
	ctx context.Context,
	j job.Uploadable,
	zname string,
) error {
	l := logutils.LoggerFromContext(ctx)

	defer func() {
		if err := os.Remove(zname); err != nil {
			l.Error("Failed to remove zip file", zap.Error(err))
		}
		dir := filepath.Dir(zname)
		if err := os.Remove(dir); err != nil {
			l.Error("Failed to remove downloads dir", zap.Error(err))
		}
	}()

	return s.uploadFromZip(ctx, j, zname)
}

func (s *Server) uploadFromZip(
	ctx context.Context,
	j job.Uploadable,
	zname string,
) error {
	errs := make([]error, 0)

	l := logutils.LoggerFromContext(ctx)

	for _, dst := range j.GetDestinations() {
		_ctx := logutils.ContextWithLogger(ctx, l.With(
			zap.String("destination_type", dst.Type),
			zap.String("destination_path", dst.Path),
			zap.String("source_path", zname),
		))

		switch dst.Type {
		case config.DestinationGcpArtifactRegistryGeneric:
			if jf := j.(job.UploadableFile); jf != nil {
				errs = append(errs,
					s.uploadFromZipToGcpArtifactRegistryGeneric(_ctx, jf, zname, dst),
				)
			}

		case config.DestinationGcpArtifactRegistryDocker:
			if jc := j.(job.UploadableContainer); jc != nil {
				errs = append(errs,
					s.uploadFromZipToGcpArtifactRegistryDocker(_ctx, jc, zname, dst),
				)
			}

		default:
			errs = append(errs,
				fmt.Errorf("unexpected destination type: %s", dst.Type),
			)
		}
	}

	return utils.FlattenErrors(errs)
}

func (s *Server) uploadFromZipToGcpArtifactRegistryGeneric(
	ctx context.Context,
	j job.UploadableFile,
	zname string,
	dst *config.Destination,
) error {
	l := logutils.LoggerFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	var artifacts *artifactregistry.ProjectsLocationsRepositoriesGenericArtifactsService
	{ // artifacts service
		_artifacts, err := s.gcp.ArtifactRegistryGeneric(ctx)
		if err != nil {
			return err
		}
		artifacts = _artifacts
	}

	var files *artifactregistry.ProjectsLocationsRepositoriesFilesService
	{ // files service
		_files, err := s.gcp.ArtifactRegistryFiles(ctx)
		if err != nil {
			return err
		}
		files = _files
	}

	z, err := zip.OpenReader(zname)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer z.Close()

	errs := make([]error, 0)
iteratingFiles:
	for _, f := range z.File {
		if f.FileInfo().IsDir() {
			continue iteratingFiles
		}

		l = l.With(
			zap.String("file", f.Name),
		)

		{ // check if the file already exists
			filter := fmt.Sprintf(`name="%s/files/%s:%s:%s"`,
				dst.Path, dst.Package, j.GetVersion(), f.Name,
			)
			res, err := files.List(dst.Path).Filter(filter).Do()
			if err == nil && res.HTTPStatusCode != http.StatusOK {
				err = fmt.Errorf("unexpected http status: %d", res.HTTPStatusCode)
			}
			if err != nil {
				errs = append(errs, err)
				l.Error("Failed to list files in gcp artifact registry",
					zap.Error(err),
					zap.String("filter", filter),
				)
				continue iteratingFiles
			}
			switch len(res.Files) {
			default:
				l.Warn("More that 1 artifact with the same name already exists in gcp artifacts registry")
				fallthrough
			case 1:
			iteratingHashes:
				for _, h := range res.Files[0].Hashes {
					switch h.Type {
					case "SHA256":
						hash, err := utils.ZipSha256(f)
						if err != nil {
							errs = append(errs, err)
							l.Error("Failed to compute sha256 hash of a file in artifact zip",
								zap.Error(err),
							)
							continue iteratingFiles
						}
						if hash == h.Value {
							l.Info("Artifact file is already uploaded, skipping...",
								zap.String("hash", "sha256:"+hash),
							)
							continue iteratingFiles
						}
						break iteratingHashes

					case "MD5":
						hash, err := utils.ZipMd5(f)
						if err != nil {
							errs = append(errs, err)
							l.Error("Failed to compute md5 hash of a file in artifact zip",
								zap.Error(err),
							)
							continue iteratingFiles
						}
						if hash == h.Value {
							l.Info("Artifact file is already uploaded, skipping...",
								zap.String("hash", "md5:"+hash),
							)
							continue iteratingFiles
						}
						break iteratingHashes

					default:
						l.Warn("Unexpected hash type",
							zap.String("hash_type", h.Type),
						)

					}
				}

				l.Info("Artifact file already exists in gcp artifact registry, but hashes don't match, overwriting...")
			case 0:
				// no-op
			}
		}

		{ // upload
			stream, err := f.Open()
			if err != nil {
				l.Error("Failed to extract artifact from the zip file",
					zap.Error(err),
				)
				errs = append(errs, err)
				continue
			}
			defer stream.Close()

			req := artifacts.Upload(dst.Path, &artifactregistry.UploadGenericArtifactRequest{
				Filename:  f.Name,
				PackageId: dst.Package,
				VersionId: j.GetVersion(),
			})
			req.Media(stream, googleapi.ContentType("application/octet-stream"))

			start := time.Now()

			l.Debug("Uploading artifact to gcp artifact registry...",
				zap.Int64("size", f.FileInfo().Size()),
			)

			res, err := req.Do()
			if err == nil && res.HTTPStatusCode != http.StatusOK {
				if res.Operation != nil && res.Operation.Error != nil {
					err = fmt.Errorf("gcp error: %s", res.Operation.Error.Message)
				} else {
					err = fmt.Errorf("unexpected http status: %d", res.HTTPStatusCode)
				}
			}
			if err != nil {
				l.Error("Failed to upload artifact to gcp artifact registry",
					zap.Error(err),
				)
				errs = append(errs, err)
			}

			l.Info("Uploaded artifact into gcp artifact registry",
				zap.Duration("duration", time.Since(start)),
				zap.Int64("size", f.FileInfo().Size()),
			)
		}
	}

	return utils.FlattenErrors(errs)
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
