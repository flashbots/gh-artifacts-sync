package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"go.uber.org/zap"
)

func (s *Server) downloadWorkflowArtifact(
	ctx context.Context,
	j *job.SyncWorkflowArtifact,
) (string, error) {
	l := logutils.LoggerFromContext(ctx)

	var downloadsDir string
	{ // create artifact downloads dir
		downloadsDir = filepath.Join(
			s.cfg.Dir.Downloads, j.RepoOwner(), j.Repo(), "workflows", strconv.Itoa(int(j.WorkflowRunID())),
		)

		if err := os.MkdirAll(downloadsDir, 0750); err != nil {
			l.Error("Failed to create artifact download directory",
				zap.Error(err),
				zap.String("path", downloadsDir),
			)
			return "", err
		}
	}

	var downloadLink string
	{ // get the download link
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_url, _, err := s.github.Actions.DownloadArtifact(
			ctx, j.RepoOwner(), j.Repo(), j.ArtifactID(), 16,
		)
		if err != nil {
			l.Error("Failed to get the download link",
				zap.Error(err),
			)
			return "", err
		}
		downloadLink = _url.String()
	}

	var fname string
	{ // download
		fname = filepath.Join(downloadsDir, j.ArtifactName())
		if err := s.downloadFromGithub(ctx, downloadLink, fname, time.Minute); err != nil {
			l.Error("Failed to download an artifact", zap.Error(err))
			return "", err
		}
	}

	return fname, nil
}

func (s *Server) downloadReleaseAsset(
	ctx context.Context,
	j *job.SyncReleaseAsset,
) (string, error) {
	l := logutils.LoggerFromContext(ctx)

	var downloadsDir string
	{ // create asset downloads dir
		downloadsDir = filepath.Join(
			s.cfg.Dir.Downloads, j.RepoOwner(), j.Repo(), "assets", strconv.Itoa(int(j.AssetID())),
		)
		if err := os.MkdirAll(downloadsDir, 0750); err != nil {
			l.Error("Failed to create asset download directory",
				zap.Error(err),
				zap.String("path", downloadsDir),
			)
			return "", err
		}
	}

	var fname string
	{ // download
		fname = filepath.Join(downloadsDir, j.AssetName())
		if err := s.downloadFromGithub(ctx, *j.Asset.URL, fname, time.Minute); err != nil {
			l.Error("Failed to download an asset", zap.Error(err))
			return "", err
		}
	}

	return fname, nil
}

func (s *Server) downloadFromGithub(
	ctx context.Context,
	url, fname string,
	timeout time.Duration,
) error {
	l := logutils.LoggerFromContext(ctx).With(
		zap.String("url", url),
		zap.String("file_name", fname),
	)

	var stream io.ReadCloser

	{ // get http stream
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create http request: %w", err)
		}
		res, err := s.github.Client().Do(req)
		if err != nil {
			return fmt.Errorf("failed to execute http request: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed b/c of an unexpected status: %d",
				res.StatusCode,
			)
		}

		stream = res.Body
	}

	{ // download
		file, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			if file != nil {
				err = errors.Join(err,
					file.Close(),
				)
			}
			return fmt.Errorf("failed to create download file: %w", err)
		}

		l.Debug("Downloading a file...")
		start := time.Now()

		if _, err = io.Copy(file, stream); err != nil {
			return fmt.Errorf("failed to download a file: %w", errors.Join(err,
				file.Close(),
			))
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close the downloaded file: %w", err)
		}

		l.Info("Downloaded a file",
			zap.Duration("duration", time.Since(start)),
		)
	}

	return nil
}
