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

	crauthn "github.com/google/go-containerregistry/pkg/authn"
	crname "github.com/google/go-containerregistry/pkg/name"
	cr "github.com/google/go-containerregistry/pkg/v1"
	crempty "github.com/google/go-containerregistry/pkg/v1/empty"
	crmutate "github.com/google/go-containerregistry/pkg/v1/mutate"
	crremote "github.com/google/go-containerregistry/pkg/v1/remote"
	crtarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

type container struct {
	config   *cr.ConfigFile
	digest   cr.Hash
	image    cr.Image
	manifest *cr.Manifest
}

func (s *Server) dockerExtractImagesAndAttestations(
	indexManifest *cr.IndexManifest,
) (images map[string]*cr.Descriptor, attestations map[string]*cr.Descriptor, err error) {
	if indexManifest == nil {
		return nil, nil, nil
	}

	images = make(map[string]*cr.Descriptor, 0)
	attestations = make(map[string]*cr.Descriptor, 0)

	errs := make([]error, 0)
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

	return images, attestations, utils.FlattenErrors(errs)
}

func (s *Server) dockerFilterIndexManifest(
	indexManifest *cr.IndexManifest,
	dst *config.Destination,
) error {
	var (
		attestations map[string]*cr.Descriptor
		images       map[string]*cr.Descriptor
		err          error
	)

	{ // separate images from their respective attestations
		images, attestations, err = s.dockerExtractImagesAndAttestations(indexManifest)
		if len(images) == 0 && len(attestations) == 0 {
			return err
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
		if attestation, ok := attestations[digest]; ok {
			indexManifest.Manifests = append(indexManifest.Manifests, *attestation)
		}
	}

	return nil
}

func (s *Server) dockerPrepareImage(
	ctx context.Context,
	j job.UploadableContainer,
	stream *zip.ReadCloser,
	dst *config.Destination,
) (
	crname.Reference, cr.Image, cr.ImageIndex, error,
) {
	l := logutils.LoggerFromContext(ctx)

	errs := make([]error, 0)

	var containers = make(map[string]*container, 0)
	var indexManifest *cr.IndexManifest
	{ // get index and images
		for _, f := range stream.File {
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
				image, err := crtarball.Image(helperZipFileOpener(f), nil)
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

	{ // filter by platform
		switch indexManifest {
		case nil:
			for originalDigest, container := range containers {
				if !dst.HasPlatform(container.config.Platform()) {
					delete(containers, originalDigest)
				}
			}

		default:
			if err := s.dockerFilterIndexManifest(indexManifest, dst); err != nil {
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
		l.Info("No matching platforms, skipping...")
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

	if indexManifest == nil || len(indexManifest.Manifests) == 1 {
		for _, c := range containers {
			// there's only 1 if there's no index
			return ref, c.image, nil, utils.FlattenErrors(errs)
		}
		return nil, nil, nil, nil
	}

	var index cr.ImageIndex = crempty.Index
	{ // prepare container index
		for _, desc := range indexManifest.Manifests {
			originalDigest := desc.Digest.String()
			container := containers[originalDigest]
			annotations := desc.Annotations

			if annotations["vnd.docker.reference.type"] == "attestation-manifest" {
				if originalReferenceDigest, ok := annotations["vnd.docker.reference.digest"]; ok {
					if reference, ok := containers[originalReferenceDigest]; ok {
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
	}
	return ref, nil, index, utils.FlattenErrors(errs)
}

func (s *Server) dockerTagRemoteSubImages(
	ctx context.Context,
	ref crname.Reference,
	auth crauthn.Authenticator,
) error {
	l := logutils.LoggerFromContext(ctx)

	desc, err := crremote.Get(ref, crremote.WithAuth(auth))
	if err != nil {
		return fmt.Errorf("failed to get a descriptor for container image: %s: %w",
			ref.Name(), err,
		)
	}

	if !desc.MediaType.IsIndex() {
		return nil
	}

	index, err := crremote.Index(ref, crremote.WithAuth(auth))
	if err != nil {
		return fmt.Errorf("failed to retrieve container index: %s: %w",
			ref.Name(), err,
		)
	}

	indexManifest, err := index.IndexManifest()
	if err != nil {
		return fmt.Errorf("failed to get image index manifest from a descriptor: %s: %s: %w",
			ref.Name(), desc.Digest.String(), err,
		)
	}

	l.Debug("Downloaded an index",
		zap.String("digest", desc.Digest.String()),
		zap.String("reference", ref.Name()),
		zap.Any("annotations", indexManifest.Annotations),
	)

	images, attestations, err := s.dockerExtractImagesAndAttestations(indexManifest)
	if len(images) == 0 && len(attestations) == 0 {
		return err
	}

	errs := make([]error, 0)
	for _, desc := range images {
		image, err := index.Image(desc.Digest)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get image from an index: %s: %s: %w",
				ref.Name(), desc.Digest.String(), err,
			))
			continue
		}

		_tag := fmt.Sprintf("%s:%s-%s-%s",
			ref.Context().Name(), ref.Identifier(), desc.Platform.OS, desc.Platform.Architecture,
		)
		tag, err := crname.NewTag(_tag)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse a tag: %s: %w",
				_tag, err,
			))
			continue
		}

		if err := crremote.Tag(tag, image, crremote.WithAuth(auth)); err != nil {
			errs = append(errs, fmt.Errorf("failed to tag sub-image: %s: %w",
				_tag, err,
			))
			continue
		}
	}

	for digest, desc := range attestations {
		reference, ok := images[digest]
		if !ok {
			continue
		}

		image, err := index.Image(desc.Digest)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get image from an index: %s: %s: %w",
				ref.Name(), desc.Digest.String(), err,
			))
			continue
		}

		_tag := fmt.Sprintf("%s:%s-%s-%s-attestation",
			ref.Context().Name(), ref.Identifier(), reference.Platform.OS, reference.Platform.Architecture,
		)
		tag, err := crname.NewTag(_tag)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse a tag: %s: %w",
				_tag, err,
			))
			continue
		}

		if err := crremote.Tag(tag, image, crremote.WithAuth(auth)); err != nil {
			errs = append(errs, fmt.Errorf("failed to tag sub-image: %s: %w",
				_tag, err,
			))
			continue
		}
	}

	return utils.FlattenErrors(errs)
}
