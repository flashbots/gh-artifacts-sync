package server

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"
	"go.uber.org/zap"
)

func (s *Server) schedulerIngestJobs(_ time.Time) {
	l := s.logger

	if count := s.jobInFlight.Load(); count > 0 {
		l.Debug("There are still jobs in-flight, skipping...",
			zap.Int64("count", count),
		)
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
		if filepath.Ext(path) != ".json" {
			// renameio firstly creates files like `.{fname}NNNNNNNNNNNNNNNNNNN`
			// that we should ignore
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

func (s *Server) schedulerHandleJob(ctx context.Context, j job.Job) {
	defer s.jobInFlight.Sub(1)

	l := logutils.LoggerFromContext(ctx).With(
		zap.String("job_id", job.ID(j)),
	)
	ctx = logutils.ContextWithLogger(ctx, l)

	var err error
	{ // dispatch the jobs
		switch j := j.(type) {
		case *job.CleanupUnparseableJob:
			err = s.handleCleanupUnparseableJob(ctx, j)

		case *job.DiscoverWorkflowArtifacts:
			err = s.handleDiscoverWorkflowArtifacts(ctx, j)

		case *job.SyncContainerRegistryPackage:
			err = s.handleSyncContainerRegistryPackage(ctx, j)

		case *job.SyncReleaseAsset:
			err = s.handleSyncRepositoryRelease(ctx, j)

		case *job.SyncWorkflowArtifact:
			err = s.handleSyncWorkflowArtifact(ctx, j)
		}
	}

	if err != nil {
		noRetryErr := &utils.NonRetryableError{}
		if errors.As(err, &noRetryErr) {
			l.Error("Non-retryable error encountered, will remove the job",
				zap.Error(noRetryErr),
			)
		} else {
			l.Warn("Retryable error encountered, keeping the job",
				zap.Error(err),
			)
			return
		}
	}

	s.RemoveJob(ctx, j)
}
