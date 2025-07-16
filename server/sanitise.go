package server

import (
	"errors"

	"github.com/google/go-github/v73/github"
)

func (s *Server) sanitiseRegistryPackageEvent(e *github.RegistryPackageEvent) error {
	if e == nil {
		return errors.New("nil registry package event")
	}

	if e.Action == nil {
		return errors.New("missing action")
	}

	if e.RegistryPackage == nil {
		return errors.New("missing registry package info")
	}

	if e.RegistryPackage.Ecosystem == nil {
		return errors.New("missing registry package ecosystem")
	}

	if e.RegistryPackage.Name == nil {
		return errors.New("missing registry package name")
	}

	if e.RegistryPackage.PackageType == nil {
		return errors.New("missing registry package type")
	}

	if e.RegistryPackage.PackageVersion == nil {
		return errors.New("missing registry package version info")
	}

	if e.RegistryPackage.PackageVersion.ContainerMetadata == nil {
		return errors.New("missing registry package container metadata info")
	}

	if e.RegistryPackage.PackageVersion.ContainerMetadata.Tag == nil {
		return errors.New("missing registry package container tag info")
	}

	if e.RegistryPackage.PackageVersion.ContainerMetadata.Tag.Digest == nil {
		return errors.New("missing registry package container digest")
	}

	if e.RegistryPackage.PackageVersion.ContainerMetadata.Tag.Name == nil {
		return errors.New("missing registry package container tag name")
	}

	if e.RegistryPackage.PackageVersion.ID == nil {
		return errors.New("missing registry package version id")
	}

	if e.RegistryPackage.PackageVersion.PackageURL == nil {
		return errors.New("missing registry package version url")
	}

	if e.RegistryPackage.PackageVersion.Version == nil {
		return errors.New("missing registry package version")
	}

	if e.Repository == nil {
		return errors.New("missing repo info")
	}

	if e.Repository.FullName == nil {
		return errors.New("missing repo full name")
	}

	if e.Repository.Name == nil {
		return errors.New("missing repo name")
	}

	if e.Repository.Owner == nil {
		return errors.New("missing repo owner")
	}

	return nil
}

func (s *Server) sanitiseReleaseEvent(e *github.ReleaseEvent) error {
	if e == nil {
		return errors.New("nil release event")
	}

	if e.Action == nil {
		return errors.New("missing action")
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

	if e.WorkflowRun.TriggeringActor == nil {
		return errors.New("missing workflow run triggering actor info")
	}

	if e.WorkflowRun.TriggeringActor.Login == nil {
		return errors.New("missing workflow run triggering actor login")
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
