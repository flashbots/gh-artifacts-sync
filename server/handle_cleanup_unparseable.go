package server

import (
	"context"
	"os"

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

	if err := os.Remove(job.Path(j)); err != nil {
		l.Error("Failed to remove unparseable job",
			zap.Error(err),
			zap.String("path", job.Path(j)),
		)
	}

	return nil
}
