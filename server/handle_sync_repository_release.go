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
		zap.String("repo_owner", j.RepoOwner()),
		zap.String("repo", j.Repo()),
		zap.Int64("asset_id", j.AssetID()),
		zap.String("asset_name", j.AssetName()),
		zap.String("version", j.Version),
	)
	ctx = logutils.ContextWithLogger(ctx, l)

	l.Info("Synchronising release asset...")

	zname, err := s.downloadReleaseAsset(ctx, j)
	if err != nil {
		return errors.Join(err, os.Remove(zname))
	}

	if err := s.uploadAndDelete(ctx, j, zname); err != nil {
		l.Error("Failed to synchronise release asset",
			zap.Error(err),
		)
		return err
	}

	l.Info("Done synchronising release asset")

	return nil
}
