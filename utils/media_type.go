package utils

import (
	cr "github.com/google/go-containerregistry/pkg/v1"
	crtypes "github.com/google/go-containerregistry/pkg/v1/types"
)

func MostFrequentMediaType(image cr.Image) (crtypes.MediaType, error) {
	layers, err := image.Layers()
	if err != nil {
		return "", err
	}

	mediaTypes := make(map[crtypes.MediaType]int)
	for _, layer := range layers {
		mediaType, err := layer.MediaType()
		if err != nil {
			return "", err
		}
		if _, exists := mediaTypes[mediaType]; !exists {
			mediaTypes[mediaType] = 0
		}
		mediaTypes[mediaType]++
	}

	var (
		mostFrequentMediaType crtypes.MediaType
		highestCount          int
	)
	for mediaType, count := range mediaTypes {
		if count > highestCount {
			highestCount = count
			mostFrequentMediaType = mediaType
		}
	}

	return mostFrequentMediaType, nil
}
