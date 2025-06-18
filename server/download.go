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

func (s *Server) download(
	ctx context.Context,
	j *job.SyncWorkflowArtifact,
) (string, error) {
	l := logutils.LoggerFromContext(ctx)

	var (
		url, zname string
		stream     io.ReadCloser
	)

	path := filepath.Join(
		s.cfg.Dir.Artifacts,
		j.RepoOwner(),
		j.Repo(),
		strconv.Itoa(int(j.WorkflowRunID())),
	)

	{ // artifact download dir
		if err := os.MkdirAll(path, 0750); err != nil {
			l.Error("Failed to create artifact download directory",
				zap.Error(err),
				zap.String("path", path),
			)
			return "", err
		}
	}

	{ // download link
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
		url = _url.String()
	}

	{ // get artifact body stream
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			l.Error("Failed to create http request",
				zap.Error(err),
			)
			return "", err
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			l.Error("Failed to execute http request",
				zap.Error(err),
				zap.String("url", url),
			)
			return "", err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			err = fmt.Errorf("download failed with status: %d", res.StatusCode)
			l.Error("Failed to download workflow artifact",
				zap.Error(err),
			)
			return "", err
		}

		stream = res.Body
	}

	{ // download zip archive
		zname = filepath.Join(path, j.ArtifactName()+".zip") // gh sends zips only
		zfile, err := os.OpenFile(zname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			l.Error("Failed to create workflow artifact zip file",
				zap.Error(err),
				zap.String("zip_file_name", zname),
			)
			return "", errors.Join(err, zfile.Close())
		}

		l.Debug("Downloading workflow artifact zip file...")
		start := time.Now()

		if _, err = io.Copy(zfile, stream); err != nil {
			l.Error("Failed to write workflow artifact zip file",
				zap.Error(err),
				zap.String("zip_file_name", zname),
			)
			return "", errors.Join(err, zfile.Close())
		}

		if err := zfile.Close(); err != nil {
			l.Error("Failed to close workflow artifacts zip file",
				zap.Error(err),
				zap.String("zip_file_name", zname),
			)
			return "", err
		}

		l.Info("Downloaded workflow artifact zip file",
			zap.String("zip_file_name", zname),
			zap.Duration("duration", time.Since(start)),
		)
	}

	return zname, nil
}
