package job

import "github.com/flashbots/gh-artifacts-sync/config"

type Uploadable interface {
	GetDestinations() []*config.Destination
}

type UploadableFile interface {
	GetDestinations() []*config.Destination
	GetVersion() string
}

type UploadableContainer interface {
	IsTagless() bool
	GetDestinations() []*config.Destination
	GetDestinationReference(*config.Destination) string
	GetTag() string
}
