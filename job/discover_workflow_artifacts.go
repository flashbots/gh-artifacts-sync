package job

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/google/go-github/v72/github"
)

const TypeDiscoverWorkflowArtifacts = "discover-workflow-artifacts"

type DiscoverWorkflowArtifacts struct {
	Meta *Meta `json:"meta"`

	WorkflowRunEvent *github.WorkflowRunEvent `json:"workflow_run_event"`
}

func NewDiscoverWorkflowArtifacts(e *github.WorkflowRunEvent) *DiscoverWorkflowArtifacts {
	var id string
	if e.WorkflowRun.ID != nil {
		id = fmt.Sprintf("%s-%d", TypeDiscoverWorkflowArtifacts, *e.WorkflowRun.ID)
	} else {
		id = fmt.Sprintf("%s-noid-%d", TypeDiscoverWorkflowArtifacts, rand.Int64())
	}

	return &DiscoverWorkflowArtifacts{
		Meta: &Meta{
			ID:   id,
			Type: TypeDiscoverWorkflowArtifacts,
		},

		WorkflowRunEvent: e,
	}
}

func (j *DiscoverWorkflowArtifacts) meta() *Meta {
	if j == nil {
		return nil
	}
	return j.Meta
}

func (j *DiscoverWorkflowArtifacts) Repo() string {
	if j == nil ||
		j.WorkflowRunEvent == nil ||
		j.WorkflowRunEvent.Repo == nil ||
		j.WorkflowRunEvent.Repo.Name == nil {
		// ---
		return ""
	}
	return *j.WorkflowRunEvent.Repo.Name
}

func (j *DiscoverWorkflowArtifacts) RepoFullName() string {
	if j == nil ||
		j.WorkflowRunEvent == nil ||
		j.WorkflowRunEvent.Repo == nil ||
		j.WorkflowRunEvent.Repo.FullName == nil {
		// ---
		return ""
	}
	return *j.WorkflowRunEvent.Repo.FullName
}

func (j *DiscoverWorkflowArtifacts) RepoOwner() string {
	if j == nil ||
		j.WorkflowRunEvent == nil ||
		j.WorkflowRunEvent.Repo == nil ||
		j.WorkflowRunEvent.Repo.Owner == nil ||
		j.WorkflowRunEvent.Repo.Owner.Login == nil {
		// ---
		return ""
	}
	return *j.WorkflowRunEvent.Repo.Owner.Login
}

func (j *DiscoverWorkflowArtifacts) WorkflowFile() string {
	if j == nil ||
		j.WorkflowRunEvent == nil ||
		j.WorkflowRunEvent.Workflow == nil ||
		j.WorkflowRunEvent.Workflow.Path == nil {
		// ---
		return ""
	}
	return strings.TrimPrefix(*j.WorkflowRunEvent.Workflow.Path, ".github/workflows/")
}

func (j *DiscoverWorkflowArtifacts) WorkflowRunID() int64 {
	if j == nil ||
		j.WorkflowRunEvent == nil ||
		j.WorkflowRunEvent.WorkflowRun == nil ||
		j.WorkflowRunEvent.WorkflowRun.ID == nil {
		// ---
		return 0
	}
	return *j.WorkflowRunEvent.WorkflowRun.ID
}
