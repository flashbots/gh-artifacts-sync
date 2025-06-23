package gcp

import (
	"context"
	"fmt"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/artifactregistry/v1"
	"google.golang.org/api/option"
)

type Client struct{}

func New() *Client {
	return &Client{}
}

func (cli *Client) AccessToken(ctx context.Context, scope string) (string, error) {
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

func (cli *Client) ArtifactRegistryGeneric(ctx context.Context) (
	*artifactregistry.ProjectsLocationsRepositoriesGenericArtifactsService, error,
) {
	creds, err := google.FindDefaultCredentials(ctx, artifactregistry.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("failed to find gcp credentials: %w", err)
	}
	svc, err := artifactregistry.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to initialise gcp artifact registry service: %w", err)
	}

	return svc.Projects.Locations.Repositories.GenericArtifacts, nil
}

func (cli *Client) ArtifactRegistryFiles(ctx context.Context) (
	*artifactregistry.ProjectsLocationsRepositoriesFilesService, error,
) {
	creds, err := google.FindDefaultCredentials(ctx, artifactregistry.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("failed to find gcp credentials: %w", err)
	}
	svc, err := artifactregistry.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to initialise gcp artifact registry service: %w", err)
	}

	return svc.Projects.Locations.Repositories.Files, nil
}
