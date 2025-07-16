package server

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"

	"go.uber.org/zap"

	crauthn "github.com/google/go-containerregistry/pkg/authn"
	crremote "github.com/google/go-containerregistry/pkg/v1/remote"
	crtransport "github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

func (s *Server) uploadFromZipToGcpArtifactRegistryDocker(
	ctx context.Context,
	j job.UploadableContainer,
	zname string,
	dst *config.Destination,
) error {
	l := logutils.LoggerFromContext(ctx)

	if j.IsTagless() {
		l.Info("Image is tag-less, skipping...")
		return nil
	}

	var z *zip.ReadCloser
	{ // open archive
		_z, err := zip.OpenReader(zname)
		if err != nil {
			return fmt.Errorf("failed to open zip file: %w", err)
		}
		defer _z.Close()
		z = _z
	}

	ref, image, index, err := s.dockerPrepareImage(ctx, j, z, dst)
	if ref == nil || (image == nil && index == nil) {
		if err != nil {
			l.Error("Failed to prepare image for upload", zap.Error(err))
		}
		return err
	} else if err != nil {
		l.Warn("There were issues while preparing image for upload", zap.Error(err))
	}

	var auth crauthn.Authenticator
	{ // get authentication token
		token, err := utils.WithTimeout(ctx, 10*time.Minute, func(ctx context.Context) (string, error) {
			return s.gcp.AccessToken(ctx, "https://www.googleapis.com/auth/cloud-platform")
		})
		if err != nil {
			l.Error("Failed to get gcp token", zap.Error(err))
			return err
		}
		auth = crauthn.FromConfig(crauthn.AuthConfig{
			Username: "oauth2accesstoken",
			Password: token,
		})
	}

	l = l.With(
		zap.String("destination_reference", ref.String()),
	)

	l.Debug("Pushing container to the destination")

	var uploadErr error
	{ // push
		switch {
		case image != nil:
			uploadErr = crremote.Write(ref, image, crremote.WithAuth(auth))

		case index != nil:
			uploadErr = crremote.WriteIndex(ref, index, crremote.WithAuth(auth))
		}
	}

	if uploadErr != nil {
		l.Error("Failed to push container image to the destination", zap.Error(uploadErr))

		transportErr := &crtransport.Error{}
		if errors.As(uploadErr, &transportErr) && !transportErr.Temporary() {
			uploadErr = utils.DoNotRetry(uploadErr)
		}

		return uploadErr
	}

	{ // tag images referred by the index at the destination
		if index != nil {
			if err := s.dockerTagRemoteSubImages(ctx, ref, auth); err != nil {
				l.Warn("Failed to tag sub-images of the container index", zap.Error(err))
			}
		}
	}

	l.Info("Pushed container image to the destination")

	return nil
}
