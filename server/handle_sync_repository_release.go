package server

import (
	"context"
	"errors"
	"os"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"go.uber.org/zap"
)

func (s *Server) handleSyncRepositoryRelease(
	ctx context.Context,
	j *job.SyncReleaseAsset,
) error {
	l := logutils.LoggerFromContext(ctx).With(
		zap.String("repo_owner", j.GetRepoOwner()),
		zap.String("repo", j.GetRepo()),
		zap.Int64("asset_id", j.GetAssetID()),
		zap.String("asset_name", j.GetAssetName()),
		zap.String("version", j.Version),
	)
	ctx = logutils.ContextWithLogger(ctx, l)

	l.Info("Synchronising release asset...")

	zname, err := s.downloadGithubReleaseAsset(ctx, j)
	if err != nil {
		l.Error("Failed to download release asset", zap.Error(err))
		return errors.Join(err, os.Remove(zname))
	}

	if err := s.uploadFromZipAndDelete(ctx, j, zname); err != nil {
		l.Error("Failed to upload release asset", zap.Error(err))
		return err
	}

	l.Info("Done synchronising release asset")

	return nil
}
