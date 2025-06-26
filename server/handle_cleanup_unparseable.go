package server

import (
	"context"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"go.uber.org/zap"
)

func (s *Server) handleCleanupUnparseableJob(
	ctx context.Context,
	j *job.CleanupUnparseableJob,
) error {
	l := logutils.LoggerFromContext(ctx)

	l.Info("Cleaning up unparseable job",
		zap.String("path", job.Path(j)),
	)

	s.RemoveJob(ctx, j)

	return nil
}
