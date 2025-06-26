package server

import (
	"context"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"
	"go.uber.org/zap"
)

func (s *Server) RemoveDownload(ctx context.Context, path string) {
	l := logutils.LoggerFromContext(ctx).With(
		zap.String("path", path),
	)

	if err := utils.SoftDelete(path, s.cfg.SoftDelete.Downloads); err != nil {
		l.Error("Failed to remove downloaded file", zap.Error(err))
		return
	}

	l.Debug("Removed a downloaded file")
}

func (s *Server) RemoveJob(ctx context.Context, j job.Job) {
	l := logutils.LoggerFromContext(ctx).With(
		zap.String("path", job.Path(j)),
	)

	if err := utils.SoftDelete(job.Path(j), s.cfg.SoftDelete.Jobs); err != nil {
		l.Error("Failed to remove persisted job", zap.Error(err))
		return
	}

	l.Debug("Removed persisted job")
}
