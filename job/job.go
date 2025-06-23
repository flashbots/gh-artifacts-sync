package job

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/google/renameio/v2"
)

type Job interface {
	meta() *Meta
}

type Meta struct {
	ID   string `json:"id"`
	Type string `json:"type"`

	persistedPath string
}

var (
	errUnknownType = errors.New("unknown job type")
)

func Load(path string) (Job, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var header struct {
		Meta Meta `json:"meta"`
	}
	if err := json.Unmarshal(bytes, &header); err != nil {
		return nil, err
	}

	var job Job

	switch header.Meta.Type {
	case TypeDiscoverWorkflowArtifacts:
		j := &DiscoverWorkflowArtifacts{}
		if err := json.Unmarshal(bytes, j); err != nil {
			return nil, err
		}
		job = j

	case TypeSyncContainerRegistryPackage:
		j := &SyncContainerRegistryPackage{}
		if err := json.Unmarshal(bytes, j); err != nil {
			return nil, err
		}
		job = j

	case TypeSyncReleaseAsset:
		j := &SyncReleaseAsset{}
		if err := json.Unmarshal(bytes, j); err != nil {
			return nil, err
		}
		job = j

	case TypeSyncWorkflowArtifact:
		j := &SyncWorkflowArtifact{}
		if err := json.Unmarshal(bytes, j); err != nil {
			return nil, err
		}
		job = j

	default:
		return nil, fmt.Errorf("%w: %s",
			errUnknownType, header.Meta.Type,
		)
	}

	job.meta().persistedPath = path

	return job, nil
}

func Save(j Job, dir string) (string, error) {
	bytes, err := json.Marshal(j)
	if err != nil {
		return "", err
	}

	fn := path.Join(dir, j.meta().ID+".json")

	if err := renameio.WriteFile(fn, bytes, 0640); err != nil {
		return "", err
	}

	return fn, nil
}

func ID(j Job) string {
	return j.meta().ID
}

func Type(j Job) string {
	return j.meta().Type
}

func Path(j Job) string {
	return j.meta().persistedPath
}
