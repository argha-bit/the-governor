package usecase

import (
	"context"

	"github.com/google/go-github/v75/github"
)

type GithubUtility interface {
	ReadFileContent(ctx context.Context, owner, repo, path string) (string, error)
	ListRepositories(ctx context.Context) ([]*github.Repository, error)
	GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error)
}

type GithubAppClient interface {
	GetClientForInstallation(installationID int64) (*github.Client, error)
}
