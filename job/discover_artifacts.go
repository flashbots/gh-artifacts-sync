package job

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/google/go-github/v72/github"
)

const TypeDiscoverArtifacts = "discover-artifacts"

type DiscoverArtifacts struct {
	Meta *Meta `json:"meta"`

	WorkflowRunEvent *github.WorkflowRunEvent `json:"workflow_run_event"`
}

func NewDiscoverArtifacts(e *github.WorkflowRunEvent) *DiscoverArtifacts {
	var id string
	if e.WorkflowRun.ID != nil {
		id = fmt.Sprintf("%s-%d", TypeDiscoverArtifacts, *e.WorkflowRun.ID)
	} else {
		id = fmt.Sprintf("%s-noid-%d", TypeDiscoverArtifacts, rand.Int64())
	}

	return &DiscoverArtifacts{
		Meta: &Meta{
			ID:   id,
			Type: TypeDiscoverArtifacts,
		},

		WorkflowRunEvent: e,
	}
}

func (j *DiscoverArtifacts) meta() *Meta {
	if j == nil {
		return nil
	}
	return j.Meta
}

func (j *DiscoverArtifacts) Owner() string {
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

func (j *DiscoverArtifacts) Repo() string {
	if j == nil ||
		j.WorkflowRunEvent == nil ||
		j.WorkflowRunEvent.Repo == nil ||
		j.WorkflowRunEvent.Repo.Name == nil {
		// ---
		return ""
	}
	return *j.WorkflowRunEvent.Repo.Name
}

func (j *DiscoverArtifacts) WorkflowFile() string {
	if j == nil ||
		j.WorkflowRunEvent == nil ||
		j.WorkflowRunEvent.Workflow == nil ||
		j.WorkflowRunEvent.Workflow.Path == nil {
		// ---
		return ""
	}
	return strings.TrimPrefix(*j.WorkflowRunEvent.Workflow.Path, ".github/workflows/")
}

func (j *DiscoverArtifacts) WorkflowRunID() int64 {
	if j == nil ||
		j.WorkflowRunEvent == nil ||
		j.WorkflowRunEvent.WorkflowRun == nil ||
		j.WorkflowRunEvent.WorkflowRun.ID == nil {
		// ---
		return 0
	}
	return *j.WorkflowRunEvent.WorkflowRun.ID
}
