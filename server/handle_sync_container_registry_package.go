package server

import (
	"context"
	"errors"
	"os"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"

	"go.uber.org/zap"
)

func (s *Server) handleSyncContainerRegistryPackage(
	ctx context.Context,
	j *job.SyncContainerRegistryPackage,
) error {
	l := logutils.LoggerFromContext(ctx).With(
		zap.String("repo", j.GetRepoFullName()),
		zap.String("package", j.GetPackageName()),
		zap.String("tag", j.GetTag()),
		zap.Int64("version_id", j.GetVersionID()),
	)
	ctx = logutils.ContextWithLogger(ctx, l)

	l.Info("Synchronising container registry package...")

	zname, err := s.downloadGithubContainerRegistryPackage(ctx, j)
	if err != nil {
		l.Error("Failed to download container registry package", zap.Error(err))
		return errors.Join(err, os.Remove(zname))
	}

	if err := s.uploadFromZipAndDelete(ctx, j, zname); err != nil {
		l.Error("Failed to upload container registry package", zap.Error(err))
		return err
	}

	return nil
}
