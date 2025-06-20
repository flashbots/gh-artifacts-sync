package server

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"go.uber.org/zap"
)

func (s *Server) scheduleJobs(_ time.Time) {
	l := s.logger

	if s.jobInFlight.Load() > 0 {
		l.Debug("There are still jobs in-flight, skipping...")
		return
	}

	err := filepath.WalkDir(s.cfg.Dir.Jobs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			l.Warn("Failure while walking the persistent dir",
				zap.Error(err),
				zap.String("persistent_dir", s.cfg.Dir.Jobs),
			)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		j, err := job.Load(path)
		if err != nil {
			j = job.NewCleanupUnparseableJob(path, err)
		}
		s.jobs <- j
		s.jobInFlight.Add(1)
		return nil
	})
	if err != nil {
		l.Error("Failed to walk the persistent dir",
			zap.Error(err),
			zap.String("persistent_dir", s.cfg.Dir.Jobs),
		)
	}
}

func (s *Server) handleJob(ctx context.Context, j job.Job) {
	defer s.jobInFlight.Sub(1)

	l := logutils.LoggerFromContext(ctx).With(
		zap.String("job_id", job.ID(j)),
	)
	ctx = logutils.ContextWithLogger(ctx, l)

	var err error
	switch j := j.(type) {
	case *job.CleanupUnparseableJob:
		err = s.handleCleanupUnparseableJob(ctx, j)

	case *job.DiscoverWorkflowArtifacts:
		err = s.handleDiscoverWorkflowArtifacts(ctx, j)

	case *job.SyncReleaseAsset:
		err = s.handleSyncRepositoryRelease(ctx, j)

	case *job.SyncWorkflowArtifact:
		err = s.handleSyncWorkflowArtifact(ctx, j)
	}

	if err != nil {
		l.Debug("Failed to handle the job",
			zap.Error(err),
		)
		return
	}

	if err := os.Remove(job.Path(j)); err != nil {
		l.Error("Failed to remove completed job",
			zap.Error(err),
			zap.String("path", job.Path(j)),
		)
	}
}
