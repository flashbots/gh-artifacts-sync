package server

import (
	"errors"

	"github.com/google/go-github/v72/github"
)

func (s *Server) sanitiseReleaseEvent(e *github.ReleaseEvent) error {
	if e == nil {
		return errors.New("nil release event")
	}

	if e.Action == nil {
		return errors.New("missing repo action")
	}

	if e.Repo == nil {
		return errors.New("missing repo info")
	}

	if e.Repo.FullName == nil {
		return errors.New("missing repo full name")
	}

	if e.Release == nil {
		return errors.New("missing release info")
	}

	if e.Release.Assets == nil {
		return errors.New("missing release assets")
	}

	for _, a := range e.Release.Assets {
		if a == nil {
			return errors.New("nil release asset")
		}

		if a.ContentType == nil {
			return errors.New("missing release asset content type")
		}

		if a.Name == nil {
			return errors.New("missing release asset name")
		}

		if a.State == nil {
			return errors.New("missing release asset state")
		}
	}

	if e.Release.Draft == nil {
		return errors.New("missing release draft marker")
	}

	if e.Release.ID == nil {
		return errors.New("missing release id")
	}

	if e.Release.Name == nil {
		return errors.New("missing release name")
	}

	if e.Release.Prerelease == nil {
		return errors.New("missing release pre-release marker")
	}

	return nil
}

func (s *Server) sanitiseWorkflowEvent(e *github.WorkflowRunEvent) error {
	if e == nil {
		return errors.New("nil workflow event")
	}

	if e.Repo == nil {
		return errors.New("missing repo info")
	}

	if e.Repo.FullName == nil {
		return errors.New("missing repo full name")
	}

	if e.Workflow == nil {
		return errors.New("missing workflow")
	}
	if e.Workflow.Path == nil {
		return errors.New("missing workflow path")
	}

	if e.WorkflowRun == nil {
		return errors.New("missing workflow run info")
	}

	if e.WorkflowRun.ID == nil {
		return errors.New("missing workflow id")
	}

	if e.WorkflowRun.Status == nil {
		return errors.New("missing workflow status")
	}

	return nil
}

func (s *Server) sanitiseArtifact(a *github.Artifact) error {
	if a == nil {
		return errors.New("nil artifact")
	}

	if a.Expired == nil {
		return errors.New("missing expiration")
	}

	if a.Name == nil {
		return errors.New("missing name")
	}

	if a.WorkflowRun == nil {
		return errors.New("missing workflow run")
	}

	if a.WorkflowRun.ID == nil {
		return errors.New("missing workflow run id")
	}

	if a.WorkflowRun.HeadSHA == nil {
		return errors.New("missing workflow run head sha")
	}

	return nil
}
