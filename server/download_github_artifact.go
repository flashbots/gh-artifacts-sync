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

func (s *Server) downloadGithubArtifact(
	ctx context.Context,
	j *job.SyncWorkflowArtifact,
) (string, error) {
	var downloadsDir string
	{ // create artifact downloads dir
		downloadsDir = filepath.Join(
			s.cfg.Dir.Downloads, j.GetRepoFullName(), "workflows", strconv.Itoa(int(j.GetWorkflowRunID())),
		)

		if err := os.MkdirAll(downloadsDir, 0750); err != nil {
			return "", fmt.Errorf("failed to create artifact download directory: %s: %w",
				downloadsDir, err,
			)
		}
	}

	var downloadLink string
	{ // get the download link
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_url, _, err := s.github.Actions.DownloadArtifact(
			ctx, j.GetRepoOwner(), j.GetRepoFullName(), j.GetArtifactID(), 16,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get the download link: %w", err)
		}
		downloadLink = _url.String()
	}

	var fname string
	{ // download
		fname = filepath.Join(downloadsDir, j.GetArtifactName())
		if err := s.downloadGithubUrl(ctx, downloadLink, fname, time.Minute); err != nil {
			return "", fmt.Errorf("failed to download an artifact: %w", err)
		}
	}

	return fname, nil
}
