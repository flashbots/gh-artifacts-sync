package server

import (
	"context"

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
		zap.String("package_url", j.GetPackageUrl()),
		zap.String("tag", j.GetTag()),
		zap.String("digest", j.GetDigest()),
		zap.Int64("version_id", j.GetVersionID()),
	)
	ctx = logutils.ContextWithLogger(ctx, l)

	l.Info("Synchronising container registry package...")

	zname, err := s.downloadGithubContainer(ctx, j)
	if err != nil {
		l.Error("Failed to download container registry package", zap.Error(err))
		s.RemoveDownload(ctx, zname)
		return err
	}

	if err := s.uploadFromZipAndDelete(ctx, j, zname); err != nil {
		l.Error("Failed to upload container registry package", zap.Error(err))
		return err
	}

	return nil
}
