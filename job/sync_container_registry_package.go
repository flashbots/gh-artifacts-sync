package job

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/google/go-github/v72/github"
)

const TypeSyncContainerRegistryPackage = "sync-container-registry-package"

type SyncContainerRegistryPackage struct {
	Meta *Meta `json:"meta"`

	Destinations []*config.Destination `json:"destinations"`
	Package      *github.Package       `json:"package"`
	Repository   *github.Repository    `json:"repository"`
}

func NewSyncContainerRegistryPackage(
	package_ *github.Package,
	repository *github.Repository,
	destinations []*config.Destination,
) *SyncContainerRegistryPackage {
	var id string
	if package_ != nil &&
		package_.PackageVersion != nil &&
		package_.PackageVersion.ID != nil {
		// ---
		id = fmt.Sprintf("%s-%d", TypeSyncContainerRegistryPackage, *package_.PackageVersion.ID)
	} else {
		id = fmt.Sprintf("%s-noid-%d", TypeSyncContainerRegistryPackage, rand.Int64())
	}

	return &SyncContainerRegistryPackage{
		Meta: &Meta{
			ID:   id,
			Type: TypeSyncContainerRegistryPackage,
		},

		Package:      package_,
		Destinations: destinations,
		Repository:   repository,
	}
}

func (j *SyncContainerRegistryPackage) meta() *Meta {
	if j == nil {
		return nil
	}
	return j.Meta
}

func (j *SyncContainerRegistryPackage) GetDestinations() []*config.Destination {
	return j.Destinations
}

func (j *SyncContainerRegistryPackage) GetDestinationReference(dst *config.Destination) string {
	tag := *j.Package.PackageVersion.ContainerMetadata.Tag.Name
	if tag == "" {
		tag = strings.ReplaceAll(
			*j.Package.PackageVersion.ContainerMetadata.Tag.Digest, ":", "-",
		)
	}
	return dst.Path + "/" + dst.Package + ":" + tag
}

func (j *SyncContainerRegistryPackage) GetDigest() string {
	if j == nil ||
		j.Package == nil ||
		j.Package.PackageVersion == nil ||
		j.Package.PackageVersion.ContainerMetadata == nil ||
		j.Package.PackageVersion.ContainerMetadata.Tag == nil ||
		j.Package.PackageVersion.ContainerMetadata.Tag.Digest == nil {
		// ---
		return ""
	}
	return *j.Package.PackageVersion.ContainerMetadata.Tag.Digest
}

func (j *SyncContainerRegistryPackage) GetPackageName() string {
	if j == nil ||
		j.Package == nil ||
		j.Package.Name == nil {
		// ---
		return ""
	}
	return *j.Package.Name
}

func (j *SyncContainerRegistryPackage) GetPackageUrl() string {
	if j == nil ||
		j.Package == nil ||
		j.Package.PackageVersion == nil ||
		j.Package.PackageVersion.PackageURL == nil {
		// ---
		return ""
	}
	if strings.HasSuffix(*j.Package.PackageVersion.PackageURL, ":") { // tag-less
		if j.Package.PackageVersion.Version == nil {
			return *j.Package.PackageVersion.PackageURL
		}
		return strings.TrimSuffix(*j.Package.PackageVersion.PackageURL, ":") + "@" + *j.Package.PackageVersion.Version
	}
	return *j.Package.PackageVersion.PackageURL
}

func (j *SyncContainerRegistryPackage) GetRepo() string {
	if j == nil ||
		j.Repository == nil ||
		j.Repository.FullName == nil {
		// ---
		return ""
	}
	parts := strings.Split(*j.Repository.FullName, "/")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func (j *SyncContainerRegistryPackage) GetRepoFullName() string {
	if j == nil ||
		j.Repository == nil ||
		j.Repository.FullName == nil {
		// ---
		return ""
	}
	return *j.Repository.FullName
}

func (j *SyncContainerRegistryPackage) GetRepoOwner() string {
	if j == nil ||
		j.Repository == nil ||
		j.Repository.FullName == nil {
		// ---
		return ""
	}
	parts := strings.Split(*j.Repository.FullName, "/")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func (j *SyncContainerRegistryPackage) GetTag() string {
	if j == nil ||
		j.Package == nil ||
		j.Package.PackageVersion == nil ||
		j.Package.PackageVersion.ContainerMetadata == nil ||
		j.Package.PackageVersion.ContainerMetadata.Tag == nil ||
		j.Package.PackageVersion.ContainerMetadata.Tag.Name == nil {
		// ---
		return ""
	}
	return *j.Package.PackageVersion.ContainerMetadata.Tag.Name
}

func (j *SyncContainerRegistryPackage) GetVersionID() int64 {
	if j == nil ||
		j.Package == nil ||
		j.Package.PackageVersion == nil ||
		j.Package.PackageVersion.ID == nil {
		// ---
		return 0
	}
	return *j.Package.PackageVersion.ID
}
