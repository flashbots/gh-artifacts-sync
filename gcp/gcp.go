package gcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/artifactregistry/v1"
	"google.golang.org/api/option"
)

type Client struct {
	artifactRegistryGeneric *artifactregistry.ProjectsLocationsRepositoriesGenericArtifactsService
	artifactRegistryFiles   *artifactregistry.ProjectsLocationsRepositoriesFilesService

	mx sync.Mutex
}

func New() *Client {
	return &Client{}
}

func (cli *Client) AccessToken(scope string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tokenSource, err := google.DefaultTokenSource(ctx, scope)
	if err != nil {
		return "", fmt.Errorf("failed to initialise gcp token source: %w", err)
	}

	token, err := tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get gcp token: %w", err)
	}

	return token.AccessToken, nil
}

func (cli *Client) ArtifactRegistryGeneric() (
	*artifactregistry.ProjectsLocationsRepositoriesGenericArtifactsService, error,
) {
	cli.mx.Lock()
	defer cli.mx.Unlock()

	if cli.artifactRegistryGeneric == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		creds, err := google.FindDefaultCredentials(ctx, artifactregistry.CloudPlatformScope)
		if err != nil {
			return nil, fmt.Errorf("failed to find gcp credentials: %w", err)
		}
		svc, err := artifactregistry.NewService(ctx, option.WithCredentials(creds))
		if err != nil {
			return nil, fmt.Errorf("failed to initialise gcp artifact registry service: %w", err)
		}

		cli.artifactRegistryGeneric = svc.Projects.Locations.Repositories.GenericArtifacts
	}

	return cli.artifactRegistryGeneric, nil
}

func (cli *Client) ArtifactRegistryFiles() (
	*artifactregistry.ProjectsLocationsRepositoriesFilesService, error,
) {
	cli.mx.Lock()
	defer cli.mx.Unlock()

	if cli.artifactRegistryFiles == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		creds, err := google.FindDefaultCredentials(ctx, artifactregistry.CloudPlatformScope)
		if err != nil {
			return nil, fmt.Errorf("failed to find gcp credentials: %w", err)
		}
		svc, err := artifactregistry.NewService(ctx, option.WithCredentials(creds))
		if err != nil {
			return nil, fmt.Errorf("failed to initialise gcp artifact registry service: %w", err)
		}

		cli.artifactRegistryFiles = svc.Projects.Locations.Repositories.Files
	}

	return cli.artifactRegistryFiles, nil
}
