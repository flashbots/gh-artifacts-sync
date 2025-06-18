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

func (s *Server) handleSyncWorkflowArtifact(
	ctx context.Context,
	j *job.SyncWorkflowArtifact,
) error {
	l := logutils.LoggerFromContext(ctx).With(
		zap.String("repo_owner", j.RepoOwner()),
		zap.String("repo", j.Repo()),
		zap.Int64("workflow_run", j.WorkflowRunID()),
		zap.Int64("artifact_id", j.ArtifactID()),
		zap.String("artifact_name", j.ArtifactName()),
	)
	ctx = logutils.ContextWithLogger(ctx, l)

	l.Info("Synchronising workflow artifact...")

	zname, err := s.download(ctx, j)
	if err != nil {
		return errors.Join(err, os.Remove(zname))
	}

	defer func() {
		if err := os.Remove(zname); err != nil {
			l.Error("Failed to remove artifact zip file",
				zap.Error(err),
				zap.String("file_name", zname),
			)
		}
		dir := filepath.Dir(zname)
		if err := os.Remove(dir); err != nil {
			l.Error("Failed to remove artifact download dir",
				zap.Error(err),
				zap.String("path", dir),
			)
		}
	}()

	if err := s.upload(ctx, j, zname); err != nil {
		l.Error("Failed to synchronise artifact",
			zap.Error(err),
		)
		return err
	}

	l.Info("Done synchronising artifact")

	return nil
}
