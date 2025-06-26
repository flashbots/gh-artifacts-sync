package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"go.uber.org/zap"
)

func (s *Server) RemoveDownload(ctx context.Context, path string) {
	l := logutils.LoggerFromContext(ctx)

	switch s.cfg.SoftDelete.Downloads {
	default:
		target := filepath.Join(s.cfg.SoftDelete.Downloads, filepath.Base(path))
		if err := os.Rename(path, target); err != nil && !errors.Is(err, os.ErrNotExist) {
			l.Error("Failed to soft-delete downloaded file, will try hard-deleting...",
				zap.Error(err),
				zap.String("path", path),
				zap.String("target", target),
			)
		}
		fallthrough

	case "":
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			l.Error("Failed to remove downloaded file",
				zap.Error(err),
				zap.String("path", path),
			)
		}
	}

	l.Debug("Removed a downloaded file",
		zap.String("path", path),
	)
}

func (s *Server) RemoveJob(ctx context.Context, j job.Job) {
	l := logutils.LoggerFromContext(ctx)

	switch s.cfg.SoftDelete.Jobs {
	default:
		target := filepath.Join(s.cfg.SoftDelete.Jobs, filepath.Base(job.Path(j)))
		if err := os.Rename(job.Path(j), target); err != nil && !errors.Is(err, os.ErrNotExist) {
			l.Error("Failed to soft-delete persisted job, will try hard-deleting...",
				zap.Error(err),
				zap.String("path", job.Path(j)),
				zap.String("target", target),
			)
		}
		fallthrough

	case "":
		if err := os.Remove(job.Path(j)); err != nil && errors.Is(err, os.ErrNotExist) {
			l.Error("Failed to remove persisted job",
				zap.Error(err),
				zap.String("path", job.Path(j)),
			)
		}
	}

	l.Debug("Removed persisted job",
		zap.String("path", job.Path(j)),
	)
}
