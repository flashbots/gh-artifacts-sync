package server

import (
	"context"
	"path/filepath"
	"time"

	"github.com/flashbots/gh-artifacts-sync/job"
	"github.com/flashbots/gh-artifacts-sync/logutils"
	"github.com/flashbots/gh-artifacts-sync/utils"
	"github.com/google/go-github/v72/github"
	"go.uber.org/zap"
)

func (s *Server) handleDiscoverWorkflowArtifacts(
	ctx context.Context,
	j *job.DiscoverWorkflowArtifacts,
) error {
	l := logutils.LoggerFromContext(ctx).With(
		zap.String("owner", j.Owner()),
		zap.String("repo", j.Repo()),
		zap.String("workflow", j.WorkflowFile()),
		zap.Int64("workflow_run_id", j.WorkflowRunID()),
	)

	l.Info("Discovering artifacts of the workflow...")

	repo, repoIsConfigured := s.cfg.Repositories[j.FullName()]
	if !repoIsConfigured {
		l.Info("Ignoring workflow b/c we don't have configuration for this repo")
		return nil
	}

	workflow, workflowIsConfigured := repo.Workflows[j.WorkflowFile()]
	if !workflowIsConfigured {
		l.Info("Ignoring workflow b/c we don't have configuration for this workflow")
		return nil
	}

	artifacts := make([]*github.Artifact, 0)
	page := 0
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		list, res, err := s.github.Actions.ListWorkflowRunArtifacts(
			ctx, j.Owner(), j.Repo(), j.WorkflowRunID(), &github.ListOptions{Page: page},
		)
		if err != nil {
			l.Error("Failed to list workflow artifacts",
				zap.Error(err),
			)
			return err
		}
		artifacts = append(artifacts, list.Artifacts...)
		if res.NextPage == 0 {
			break
		}
		page = res.NextPage
	}

	errs := make([]error, 0)

	for _, ghArtifact := range artifacts {
		if err := s.sanitiseArtifact(ghArtifact); err != nil {
			l.Warn("Invalid workflow artifact, skipping...",
				zap.Error(err),
			)
			continue
		}

		if *ghArtifact.Expired {
			l.Info("Workflow artifact expired, skipping...",
				zap.String("artifact", must(ghArtifact.Name)),
			)
			continue
		}

		for _, cfgArtifact := range workflow.Artifacts {
			var version string

			matches := cfgArtifact.Regexp().FindStringSubmatch(
				filepath.Base(*ghArtifact.Name),
			)
			if len(matches) == 0 {
				continue
			}

			if len(matches) > 1 {
				version = matches[1]
			} else {
				version = *ghArtifact.WorkflowRun.HeadSHA
			}

			j := job.NewSyncWorkflowArtifact(
				ghArtifact,
				version,
				cfgArtifact.Destinations,
				j.WorkflowRunEvent.WorkflowRun,
			)

			if fname, err := job.Save(j, s.cfg.Dir.Jobs); err == nil {
				l.Info("Persisted job",
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

	l.Info("Done discovering artifacts of the workflow")

	return utils.FlattenErrors(errs)
}
