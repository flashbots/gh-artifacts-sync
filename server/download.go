package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/flashbots/gh-artifacts-sync/logutils"

	"go.uber.org/zap"
)

func (s *Server) downloadGithubUrl(
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
		req.Header.Add("accept", "application/octet-stream")
		res, err := s.github.Client().Do(req)
		if err == nil && res.StatusCode != http.StatusOK {
			err = fmt.Errorf("unexpected http status: %d", res.StatusCode)
		}
		if err != nil {
			return fmt.Errorf("failed to execute http request: %w", err)
		}
		if res != nil {
			defer res.Body.Close()
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
		defer file.Close()

		l.Debug("Downloading a file...")
		start := time.Now()

		if _, err = io.Copy(file, stream); err != nil {
			return fmt.Errorf("failed to download a file: %w", err)
		}

		l.Info("Downloaded a file",
			zap.Duration("duration", time.Since(start)),
		)
	}

	return nil
}
