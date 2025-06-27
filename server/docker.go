package server

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"
	"go.uber.org/zap"

	crname "github.com/google/go-containerregistry/pkg/name"
	cr "github.com/google/go-containerregistry/pkg/v1"
	crempty "github.com/google/go-containerregistry/pkg/v1/empty"
	crmutate "github.com/google/go-containerregistry/pkg/v1/mutate"
	crtarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

type container struct {
	config   *cr.ConfigFile
	digest   cr.Hash
	image    cr.Image
	manifest *cr.Manifest
}

func (s *Server) prepareIndexManifestForDestination(
	indexManifest *cr.IndexManifest,
	dst *config.Destination,
) error {
	if indexManifest == nil || dst == nil {
		return nil
	}

	var (
		attestations = make(map[string]*cr.Descriptor, 0)
		images       = make(map[string]*cr.Descriptor, 0)
		errs         = make([]error, 0)
	)

	{ // separate images from their respective attestations
		for _, desc := range indexManifest.Manifests {
			if desc.Annotations["vnd.docker.reference.type"] != "attestation-manifest" {
				images[desc.Digest.String()] = &desc
				continue
			}

			digest := desc.Annotations["vnd.docker.reference.digest"]
			if digest == "" {
				err := fmt.Errorf("index contains reference w/o digest: %s",
					desc.Digest.String(),
				)
				errs = append(errs, err)
				continue
			}

			if another, collision := attestations[digest]; collision {
				err := fmt.Errorf("index contains multiple attestations for the same reference: %s: %s vs. %s",
					digest, desc.Digest.String(), another.Digest.String(),
				)
				errs = append(errs, err)
				continue
			}

			attestations[digest] = &desc
		}
	}

	{ // filter out by platform
		for digest, image := range images {
			if !dst.HasPlatform(image.Platform) {
				delete(images, digest)
				delete(attestations, digest)
			}
		}
	}

	indexManifest.Manifests = make([]cr.Descriptor, 0, len(images)+len(attestations))
	for digest, image := range images {
		indexManifest.Manifests = append(indexManifest.Manifests, *image)
		indexManifest.Manifests = append(indexManifest.Manifests, *attestations[digest])
	}

	return utils.FlattenErrors(errs)
}

func (s *Server) prepareImageForUpload(
	ctx context.Context,
	j job.UploadableContainer,
	zname string,
	dst *config.Destination,
) (
	crname.Reference, cr.Image, cr.ImageIndex, error,
) {
	l := logutils.LoggerFromContext(ctx)

	var z *zip.ReadCloser
	{ // open archive
		_z, err := zip.OpenReader(zname)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to open zip file: %w", err)
		}
		defer _z.Close()
		z = _z
	}

	errs := make([]error, 0)

	var containers = make(map[string]*container, 0)
	var indexManifest *cr.IndexManifest
	{ // get index and images
		for _, f := range z.File {
			if f.FileInfo().IsDir() {
				continue
			}

			l := l.With(
				zap.String("file", f.Name),
			)

			switch filepath.Ext(f.Name) {
			case ".json":
				stream, err := f.Open()
				if err != nil {
					l.Error("Failed to open index json", zap.Error(err))
					errs = append(errs, err)
					continue
				}

				indexManifest = &cr.IndexManifest{}
				if err := json.NewDecoder(stream).Decode(indexManifest); err != nil {
					l.Error("Failed to decode index json", zap.Error(err))
					errs = append(errs, err)
					continue
				}

			case ".tar":
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

				// digest changes depending on the compression used
				originalDigest := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(f.Name), filepath.Ext(f.Name)), "-", ":")

				containers[originalDigest] = &container{
					config:   config,
					digest:   digest,
					image:    image,
					manifest: manifest,
				}
			}
		}
	}

	{ // filter platforms
		switch indexManifest {
		case nil:
			for originalDigest, container := range containers {
				if !dst.HasPlatform(container.config.Platform()) {
					delete(containers, originalDigest)
				}
			}

		default:
			if err := s.prepareIndexManifestForDestination(indexManifest, dst); err != nil {
				errs = append(errs, err)
			}
			_containers := make(map[string]*container)
			for _, desc := range indexManifest.Manifests {
				if c := containers[desc.Digest.String()]; c != nil {
					_containers[desc.Digest.String()] = c
				}
			}
			containers = _containers
		}
	}
	if len(containers) == 0 {
		l.Debug("No matching platforms, skipping...")
		return nil, nil, nil, utils.FlattenErrors(errs)
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
			return nil, nil, nil, utils.FlattenErrors(errs)
		}
		ref = _ref
	}

	switch indexManifest {
	case nil:
		for _, c := range containers {
			// there's only 1 if there's no index
			return ref, c.image, nil, utils.FlattenErrors(errs)
		}

	default:
		var index cr.ImageIndex = crempty.Index
		for _, desc := range indexManifest.Manifests {
			originalDigest := desc.Digest.String()
			container := containers[originalDigest]
			annotations := desc.Annotations

			if annotations["vnd.docker.reference.type"] == "attestation-manifest" {
				if annotationOriginalDigest, ok := annotations["vnd.docker.reference.digest"]; ok {
					if reference, ok := containers[annotationOriginalDigest]; ok {
						annotations["vnd.docker.reference.digest"] = reference.digest.String()
					}
				}
			}

			index = crmutate.AppendManifests(index, crmutate.IndexAddendum{
				Add: container.image,

				Descriptor: cr.Descriptor{
					Annotations: annotations,
					Digest:      container.digest,
					Platform:    container.config.Platform(),
				},
			})
		}

		return ref, nil, index, utils.FlattenErrors(errs)
	}

	return nil, nil, nil, utils.FlattenErrors(errs)
}
