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

	ref, image, index, err := s.prepareImageForUpload(ctx, j, z, dst)
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
			uploadErr = utils.NoRetry(uploadErr)
		}

		return uploadErr
	}

	l.Info("Pushed container image to the destination")

	return nil
}

func zipFileOpener(zf *zip.File) crtarball.Opener {
	return func() (io.ReadCloser, error) {
		return zf.Open()
	}
}
