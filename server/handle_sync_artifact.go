package server

import (
	"context"
	"errors"
	"os"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"go.uber.org/zap"
)

func (s *Server) handleSyncWorkflowArtifact(
	ctx context.Context,
	j *job.SyncWorkflowArtifact,
) error {
	l := logutils.LoggerFromContext(ctx).With(
		zap.String("repo", j.GetRepoFullName()),
		zap.Int64("workflow_run_id", j.GetWorkflowRunID()),
		zap.Int64("artifact_id", j.GetArtifactID()),
		zap.String("artifact_name", j.GetArtifactName()),
		zap.String("version", j.Version),
	)
	ctx = logutils.ContextWithLogger(ctx, l)

	l.Info("Synchronising workflow artifact...")

	zname, err := s.downloadGithubWorkflowArtifact(ctx, j)
	if err != nil {
		l.Error("Failed to download workflow artifact", zap.Error(err))
		return errors.Join(err, os.Remove(zname))
	}

	if err := s.uploadFromZipAndDelete(ctx, j, zname); err != nil {
		l.Error("Failed to upload workflow artifact", zap.Error(err))
		return err
	}

	l.Info("Done synchronising workflow artifact")

	return nil
}
