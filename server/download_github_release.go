package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/flashbots/gh-artifacts-sync/job"
)

func (s *Server) downloadGithubReleaseAsset(
	ctx context.Context,
	j *job.SyncReleaseAsset,
) (string, error) {
	var downloadsDir string
	{ // create asset downloads dir
		downloadsDir = filepath.Join(
			s.cfg.Dir.Downloads, j.GetRepoOwner(), j.GetRepo(), "assets", strconv.Itoa(int(j.GetAssetID())),
		)
		if err := os.MkdirAll(downloadsDir, 0750); err != nil {
			return "", fmt.Errorf("failed to create asset download directory: %s: %w",
				downloadsDir, err,
			)
		}
	}

	var fname string
	{ // download
		fname = filepath.Join(downloadsDir, j.GetAssetName())
		if err := s.downloadGithubUrl(ctx, *j.Asset.URL, fname, time.Minute); err != nil {
			return "", fmt.Errorf("failed to download an asset: %w", err)
		}
	}

	return fname, nil
}
