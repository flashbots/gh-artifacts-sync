package server

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"
	"github.com/google/go-github/v72/github"
	"go.uber.org/zap"
)

func (s *Server) webhook(w http.ResponseWriter, r *http.Request) {
	l := logutils.LoggerFromRequest(r)

	if r.Method == http.MethodGet {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	payload, err := github.ValidatePayload(r, []byte(s.cfg.Github.WebhookSecret))
	if err != nil {
		l.Warn("Failed to validate webhook payload",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		l.Warn("Failed to parse webhook event",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	l.Debug("Received webhook event",
		zap.String("event_type", reflect.TypeOf(event).String()),
		zap.Any("event", event),
	)

	switch e := event.(type) {
	default:
		l.Info("Ignoring unsupported event",
			zap.String("event_type", reflect.TypeOf(event).String()),
		)
		w.WriteHeader(http.StatusOK)
		return

	case *github.RegistryPackageEvent:
		err = s.processRegistryPackageEvent(r.Context(), e)

	case *github.ReleaseEvent:
		err = s.processReleaseEvent(r.Context(), e)

	case *github.WorkflowRunEvent:
		err = s.processWorkflowEvent(r.Context(), e)

	}

	if err != nil {
		l.Error("Failed to process event",
			zap.Error(err),
			zap.Any("event", event),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) processRegistryPackageEvent(ctx context.Context, e *github.RegistryPackageEvent) error {
	l := logutils.LoggerFromContext(ctx)

	if err := s.sanitiseRegistryPackageEvent(e); err != nil {
		l.Warn("Ignoring invalid registry package event",
			zap.Error(err),
		)
		return err
	}

	l = l.With(
		zap.String("repo", *e.Repository.FullName),
		zap.String("package", *e.RegistryPackage.Name),
		zap.String("version", *e.RegistryPackage.PackageVersion.Version),
		zap.Int64("version_id", *e.RegistryPackage.PackageVersion.ID),
	)

	if *e.Action != "published" {
		l.Debug("Ignoring registry package event b/c its status is not 'published'",
			zap.String("action", *e.Action),
		)
		return nil
	}

	if *e.RegistryPackage.Ecosystem != "CONTAINER" {
		l.Debug("Ignoring registry package event b/c its ecosystem is not supported",
			zap.String("ecosystem", *e.RegistryPackage.Ecosystem),
		)
		return nil
	}

	repo, repoIsConfigured := s.cfg.Repositories[must(e.Repository.FullName)]
	if !repoIsConfigured {
		l.Debug("Ignoring registry package event b/c we don't have configuration for this repo")
		return nil
	}

	container, containerIsConfigured := repo.Containers[*e.RegistryPackage.Name]
	if !containerIsConfigured {
		l.Debug("Ignoring registry package event b/c we don't have configuration for this container")
		return nil
	}

	j := job.NewSyncContainerRegistryPackage(
		e.RegistryPackage,
		e.Repository,
		container.Destinations,
	)

	fname, err := job.Save(j, s.cfg.Dir.Jobs)
	if err != nil {
		l.Error("Failed to persist a job",
			zap.Error(err),
		)
		return err
	}

	l.Info("Persisted a job",
		zap.String("job", fname),
	)

	return nil
}

func (s *Server) processReleaseEvent(ctx context.Context, e *github.ReleaseEvent) error {
	l := logutils.LoggerFromContext(ctx)

	if err := s.sanitiseReleaseEvent(e); err != nil {
		l.Warn("Ignoring invalid release event",
			zap.Error(err),
		)
		return err
	}

	l = l.With(
		zap.String("repo", *e.Repo.FullName),
		zap.String("release", *e.Release.Name),
		zap.Int64("release_id", *e.Release.ID),
	)

	if *e.Action != "published" {
		l.Debug("Ignoring release event b/c its status is not 'published'",
			zap.String("action", *e.Action),
		)
		return nil
	}

	repo, repoIsConfigured := s.cfg.Repositories[must(e.Repo.FullName)]
	if !repoIsConfigured {
		l.Debug("Ignoring release event b/c we don't have configuration for this repo")
		return nil
	}

	errs := make([]error, 0)

	jobsCount := 0

	for _, cfgRelease := range repo.Releases {
		cfgReleaseMatches := cfgRelease.Regexp().FindStringSubmatch(*e.Release.Name)
		if len(cfgReleaseMatches) == 0 {
			continue
		}

		releaseVersion := cfgReleaseMatches[0]
		if len(cfgReleaseMatches) > 1 {
			releaseVersion = cfgReleaseMatches[1]
		}

		for _, cfgAsset := range cfgRelease.Assets {
			for _, ghAsset := range e.Release.Assets {
				cfgAssetMatches := cfgAsset.Regexp().FindStringSubmatch(*ghAsset.Name)
				if len(cfgAssetMatches) == 0 {
					continue
				}

				version := releaseVersion
				if len(cfgAssetMatches) > 1 {
					version = cfgAssetMatches[1]
				}

				if *ghAsset.State != "uploaded" {
					l.Warn("Ignoring asset b/c its state is not 'uploaded'",
						zap.String("asset", *ghAsset.Name),
						zap.String("state", *ghAsset.State),
					)
					continue
				}

				if *ghAsset.ContentType != "application/zip" {
					l.Warn("Ignoring asset b/c it's not a zip archive",
						zap.String("asset", *ghAsset.Name),
						zap.String("content_type", *ghAsset.ContentType),
					)
					continue
				}

				j := job.NewSyncReleaseAsset(
					ghAsset,
					version,
					cfgAsset.Destinations,
				)
				jobsCount++

				if fname, err := job.Save(j, s.cfg.Dir.Jobs); err == nil {
					l.Info("Persisted a job",
						zap.String("job", fname),
					)
				} else {
					l.Error("Failed to persist a job",
						zap.Error(err),
					)
					errs = append(errs, err)
				}
			}
		}
	}

	if jobsCount == 0 {
		l.Debug("Ignoring release event b/c we don't have release/asset matches")
	}

	return utils.FlattenErrors(errs)
}

func (s *Server) processWorkflowEvent(ctx context.Context, e *github.WorkflowRunEvent) error {
	l := logutils.LoggerFromContext(ctx)

	if err := s.sanitiseWorkflowEvent(e); err != nil {
		l.Warn("Ignoring invalid workflow event",
			zap.Error(err),
		)
		return err
	}

	l = l.With(
		zap.String("repo", *e.Repo.FullName),
		zap.String("workflow", strings.TrimPrefix(*e.Workflow.Path, ".github/workflows/")),
		zap.Int64("workflow_id", *e.WorkflowRun.ID),
	)

	if *e.WorkflowRun.Status != "completed" {
		l.Debug("Ignoring workflow event b/c its status is not 'completed'",
			zap.String("status", *e.WorkflowRun.Status),
		)
		return nil
	}

	conclusion := must(e.WorkflowRun.Conclusion)
	if conclusion != "success" {
		l.Debug("Ignoring workflow event b/c its conclusion is not 'success'",
			zap.String("conclusion", conclusion),
		)
		return nil
	}

	repo, repoIsConfigured := s.cfg.Repositories[must(e.Repo.FullName)]
	if !repoIsConfigured {
		l.Debug("Ignoring workflow event b/c we don't have configuration for this repo")
		return nil
	}

	workflow := strings.TrimPrefix(must(e.WorkflowRun.Path), ".github/workflows/")
	if _, workflowIsConfigured := repo.Workflows[workflow]; !workflowIsConfigured {
		l.Debug("Ignoring workflow event b/c we don't have configuration for this workflow")
		return nil
	}

	fname, err := job.Save(job.NewDiscoverWorkflowArtifacts(e), s.cfg.Dir.Jobs)
	if err != nil {
		l.Error("Failed to persist a job",
			zap.Error(err),
		)
	}

	l.Info("Persisted a job",
		zap.String("job", fname),
	)

	return nil
}

func must(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}
