package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"

	"go.uber.org/zap"
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
