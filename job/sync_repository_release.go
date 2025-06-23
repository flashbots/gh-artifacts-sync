package job

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/google/go-github/v72/github"
)

const TypeSyncReleaseAsset = "sync-release-asset"

type SyncReleaseAsset struct {
	Meta *Meta `json:"meta"`

	Asset        *github.ReleaseAsset  `json:"asset"`
	Destinations []*config.Destination `json:"destinations"`
	Version      string                `json:"version"`
}

func NewSyncReleaseAsset(
	asset *github.ReleaseAsset,
	version string,
	destinations []*config.Destination,
) *SyncReleaseAsset {
	var id string
	if asset != nil &&
		asset.ID != nil {
		// ---
		id = fmt.Sprintf("%s-%d", TypeSyncReleaseAsset, *asset.ID)
	} else {
		id = fmt.Sprintf("%s-noid-%d", TypeSyncReleaseAsset, rand.Int64())
	}

	return &SyncReleaseAsset{
		Meta: &Meta{
			ID:   id,
			Type: TypeSyncReleaseAsset,
		},

		Asset:        asset,
		Destinations: destinations,
		Version:      version,
	}
}

func (j *SyncReleaseAsset) meta() *Meta {
	if j == nil {
		return nil
	}
	return j.Meta
}

func (j *SyncReleaseAsset) GetAssetID() int64 {
	if j == nil ||
		j.Asset == nil ||
		j.Asset.ID == nil {
		// ---
		return 0
	}
	return *j.Asset.ID
}

func (j *SyncReleaseAsset) GetAssetName() string {
	if j == nil ||
		j.Asset == nil ||
		j.Asset.Name == nil {
		// ---
		return ""
	}
	return *j.Asset.Name
}

func (j *SyncReleaseAsset) GetDestinations() []*config.Destination {
	return j.Destinations
}

func (j *SyncReleaseAsset) GetRepo() string {
	if j == nil ||
		j.Asset == nil ||
		j.Asset.URL == nil {
		// ---
		return ""
	}
	parts := strings.Split(
		strings.TrimPrefix(*j.Asset.URL, "https://api.github.com/repos/"),
		"/",
	)
	if len(parts) != 5 {
		return ""
	}
	return parts[1]
}

func (j *SyncReleaseAsset) GetRepoOwner() string {
	if j == nil ||
		j.Asset == nil ||
		j.Asset.URL == nil {
		// ---
		return ""
	}
	parts := strings.Split(
		strings.TrimPrefix(*j.Asset.URL, "https://api.github.com/repos/"),
		"/",
	)
	if len(parts) != 5 {
		return ""
	}
	return parts[0]
}

func (j *SyncReleaseAsset) GetVersion() string {
	return j.Version
}
