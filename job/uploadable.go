package job

import "github.com/flashbots/gh-artifacts-sync/config"

type Uploadable interface {
	GetDestinations() []*config.Destination
	GetVersion() string
}
