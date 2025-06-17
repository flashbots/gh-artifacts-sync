package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/flashbots/gh-artifacts-sync/httplogger"
	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"
	"github.com/google/go-github/v72/github"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type Server struct {
	cfg *config.Config

	failure chan error

	github *github.Client
	logger *zap.Logger
	server *http.Server
	ticker *time.Ticker

	jobs        chan job.Job
	jobInFlight *atomic.Int64
}

func New(cfg *config.Config) (*Server, error) {
	s := &Server{
		cfg:         cfg,
		failure:     make(chan error, 1),
		jobs:        make(chan job.Job, 100),
		jobInFlight: atomic.NewInt64(0),
		logger:      zap.L(),
		ticker:      time.NewTicker(5 * time.Second),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.webhook)
	mux.Handle("/metrics", promhttp.Handler())
	handler := httplogger.Middleware(s.logger, mux)

	s.server = &http.Server{
		Addr:              cfg.Server.ListenAddress,
		ErrorLog:          logutils.NewHttpServerErrorLogger(s.logger),
		Handler:           handler,
		MaxHeaderBytes:    1024,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	transport, err := ghinstallation.New(
		http.DefaultTransport,
		cfg.Github.App.ID,
		cfg.Github.App.InstallationID,
		[]byte(cfg.Github.App.PrivateKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise github app: %w", err)
	}
	s.github = github.NewClient(&http.Client{
		Transport: transport,
	})

	return s, nil
}

func (s *Server) Run() error {
	l := s.logger
	ctx := logutils.ContextWithLogger(context.Background(), l)

	go func() {
		for j := range s.jobs {
			s.handleJob(ctx, j)
		}
	}()

	go func() { // run the job ticker
		for {
			s.scheduleJobs(<-s.ticker.C)
		}
	}()

	go func() { // run the server
		l.Info("Github artifacts sync server is going up...",
			zap.String("server_listen_address", s.cfg.Server.ListenAddress),
		)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.failure <- err
		}
		l.Info("Github artifacts sync server is down")
	}()

	errs := []error{}
	{ // wait until termination or internal failure
		terminator := make(chan os.Signal, 1)
		signal.Notify(terminator, os.Interrupt, syscall.SIGTERM)

		select {
		case stop := <-terminator:
			l.Info("Stop signal received; shutting down...",
				zap.String("signal", stop.String()),
			)
		case err := <-s.failure:
			l.Error("Internal failure; shutting down...",
				zap.Error(err),
			)
			errs = append(errs, err)
		exhaustErrors:
			for { // exhaust the errors
				select {
				case err := <-s.failure:
					l.Error("Extra internal failure",
						zap.Error(err),
					)
					errs = append(errs, err)
				default:
					break exhaustErrors
				}
			}
		}
	}

	{ // stop the server
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			l.Error("Github artifacts sync server shutdown failed",
				zap.Error(err),
			)
		}
	}

	{ // stop the job ticker
		s.ticker.Stop()
	}

	close(s.jobs)

	return utils.FlattenErrors(errs)
}

func (s *Server) Config() *config.Config {
	return s.cfg
}

func (s *Server) Github() *github.Client {
	return s.github
}
