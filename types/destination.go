package types

type Destination string

const (
	DestinationGcpArtifactRegistryGeneric Destination = "gcp.artifactregistry.generic"
)

var Destinations = []Destination{
	DestinationGcpArtifactRegistryGeneric,
}
