package server

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
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
		l.Warn("Failed to validate payload",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		l.Warn("Failed to parse webhook",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *github.WorkflowRunEvent:
		if err := s.processWorkflowEvent(r.Context(), e); err != nil {
			l.Error("Failed to process workflow event",
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	default:
		l.Info("Ignoring event",
			zap.String("event_type", reflect.TypeOf(event).String()),
		)
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (s *Server) processWorkflowEvent(ctx context.Context, e *github.WorkflowRunEvent) error {
	l := logutils.LoggerFromContext(ctx)

	if err := s.sanitiseWorkflowEvent(e); err != nil {
		l.Info("Ignoring workflow event",
			zap.Error(err),
		)
		return err
	}

	l = l.With(
		zap.String("repo", *e.Repo.FullName),
		zap.String("workflow", strings.TrimPrefix(*e.Workflow.Path, ".github/workflows/")),
		zap.Int64("workflow_id", *e.WorkflowRun.ID),
	)

	status := must(e.WorkflowRun.Status)
	if status != "completed" {
		l.Debug("Ignoring workflow event b/c status is not 'completed'",
			zap.String("status", status),
		)
		return nil
	}

	conclusion := must(e.WorkflowRun.Conclusion)
	if conclusion != "success" {
		l.Debug("Ignoring workflow event b/c conclusion is not 'success'",
			zap.Int64("workflow_id", *e.WorkflowRun.ID),
			zap.String("conclusion", conclusion),
		)
		return nil
	}

	workflows, repoIsConfigured := s.cfg.Harvest[must(e.Repo.FullName)]
	if !repoIsConfigured {
		l.Debug("Ignoring workflow event b/c we don't have configuration for this repo")
		return nil
	}

	workflow := strings.TrimPrefix(must(e.WorkflowRun.Path), ".github/workflows/")
	if _, workflowIsConfigured := workflows[workflow]; !workflowIsConfigured {
		l.Debug("Ignoring workflow event b/c we don't have configuration for this workflow",
			zap.Int64("workflow_id", *e.WorkflowRun.ID),
			zap.String("repo", must(e.Repo.FullName)),
			zap.String("workflow", workflow),
		)
		return nil
	}

	fname, err := job.Save(job.NewDiscoverArtifacts(e), s.cfg.Dir.Jobs)
	if err != nil {
		l.Error("Failed to persist discover-artifacts job",
			zap.Error(err),
		)
	}

	l.Info("Persisted discover-artifacts job",
		zap.Int64("workflow_id", *e.WorkflowRun.ID),
		zap.String("repo", must(e.Repo.FullName)),
		zap.String("workflow", must(e.WorkflowRun.Path)),
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
