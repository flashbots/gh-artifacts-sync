package job

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/google/go-github/v72/github"
)

const TypeSyncArtifact = "sync-artifact"

type SyncArtifact struct {
	Meta *Meta `json:"meta"`

	Artifact     *github.Artifact      `json:"artifact"`
	Version      string                `json:"version"`
	Destinations []*config.Destination `json:"destinations"`
	WorkflowRun  *github.WorkflowRun   `json:"workflow_run"`
}

func NewSyncArtifact(
	artifact *github.Artifact,
	version string,
	destinations []*config.Destination,
	workflowRun *github.WorkflowRun,
) *SyncArtifact {
	var id string
	if artifact != nil &&
		artifact.ID != nil &&
		artifact.WorkflowRun != nil &&
		artifact.WorkflowRun.ID != nil {
		// ---
		id = fmt.Sprintf("%s-%d-%d", TypeSyncArtifact, *artifact.WorkflowRun.ID, *artifact.ID)
	} else {
		id = fmt.Sprintf("%s-noid-noid-%d", TypeSyncArtifact, rand.Int64())
	}

	return &SyncArtifact{
		Meta: &Meta{
			ID:   id,
			Type: TypeSyncArtifact,
		},

		Artifact:     artifact,
		Destinations: destinations,
		Version:      version,
		WorkflowRun:  workflowRun,
	}
}

func (j *SyncArtifact) meta() *Meta {
	if j == nil {
		return nil
	}
	return j.Meta
}

func (j *SyncArtifact) ArtifactName() string {
	if j == nil ||
		j.Artifact == nil ||
		j.Artifact.Name == nil {
		// ---
		return ""
	}
	return *j.Artifact.Name
}

func (j *SyncArtifact) ArtifactID() int64 {
	if j == nil ||
		j.Artifact == nil ||
		j.Artifact.ID == nil {
		return 0
	}
	return *j.Artifact.ID
}

func (j *SyncArtifact) RepoOwner() string {
	if j == nil ||
		j.Artifact == nil ||
		j.Artifact.URL == nil {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(*j.Artifact.URL, "https://api.github.com/repos/"), "/")
	if len(parts) != 5 {
		return ""
	}
	return parts[0]
}

func (j *SyncArtifact) Repo() string {
	if j == nil ||
		j.Artifact == nil ||
		j.Artifact.URL == nil {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(*j.Artifact.URL, "https://api.github.com/repos/"), "/")
	if len(parts) != 5 {
		return ""
	}
	return parts[1]
}

func (j *SyncArtifact) URL() string {
	if j == nil ||
		j.Artifact == nil ||
		j.Artifact.URL == nil {
		// ---
		return ""
	}
	return *j.Artifact.URL
}

func (j *SyncArtifact) WorkflowRunID() int64 {
	if j == nil ||
		j.Artifact == nil ||
		j.Artifact.WorkflowRun == nil ||
		j.Artifact.WorkflowRun.ID == nil {
		// ---
		return 0
	}
	return *j.Artifact.WorkflowRun.ID
}
