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
		err := os.Rename(path, target)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			return
		}
		l.Error("Failed to soft-delete downloaded file, will try hard-deleting...",
			zap.Error(err),
			zap.String("path", path),
			zap.String("target", target),
		)
		fallthrough

	case "":
		err := os.Remove(path)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			return
		}
		l.Error("Failed to remove downloaded file",
			zap.Error(err),
			zap.String("path", path),
		)
	}
}

func (s *Server) RemoveJob(ctx context.Context, j job.Job) {
	l := logutils.LoggerFromContext(ctx)

	switch s.cfg.SoftDelete.Jobs {
	default:
		target := filepath.Join(s.cfg.SoftDelete.Jobs, filepath.Base(job.Path(j)))
		err := os.Rename(job.Path(j), target)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			return
		}
		l.Error("Failed to soft-delete persisted job, will try hard-deleting...",
			zap.Error(err),
			zap.String("path", job.Path(j)),
			zap.String("target", target),
		)
		fallthrough

	case "":
		err := os.Remove(job.Path(j))
		if err == nil || errors.Is(err, os.ErrNotExist) {
			return
		}
		l.Error("Failed to remove persisted job",
			zap.Error(err),
			zap.String("path", job.Path(j)),
		)
	}
}
