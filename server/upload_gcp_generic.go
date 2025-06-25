package server

import (
	"archive/zip"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"

	"go.uber.org/zap"

	"google.golang.org/api/artifactregistry/v1"
	"google.golang.org/api/googleapi"
)

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
