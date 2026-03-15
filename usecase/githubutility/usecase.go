package githubutility

import (
	"context"
	"fmt"
	"net/http"
	"the_governor/usecase"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v75/github"
)

type GitHubAppClient struct {
	appID          int64
	privateKeyPath string
}

type GitHubUtility struct {
	*github.Client
}

func NewGitHubAppClient(appID int64, privateKeyPath string) usecase.GithubAppClient {
	return &GitHubAppClient{
		appID:          appID,
		privateKeyPath: privateKeyPath,
	}
}

func (g *GitHubUtility) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	repoObj, _, err := g.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	return repoObj, nil
}

func (g *GitHubUtility) ListRepositories(ctx context.Context) ([]*github.Repository, error) {
	opts := &github.ListOptions{PerPage: 100}
	var allRepos []*github.Repository

	for {
		repos, resp, err := g.Apps.ListRepos(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}

		allRepos = append(allRepos, repos.Repositories...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}
func (g *GitHubUtility) ReadFileContent(ctx context.Context, owner, repo, path string) (string, error) {
	fileContent, _, _, err := g.Repositories.GetContents(
		ctx,
		owner,
		repo,
		path,
		&github.RepositoryContentGetOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %w", err)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode content: %w", err)
	}

	return content, nil
}

func NewGitHubUtility(githubAppClient usecase.GithubAppClient, installationId int64) usecase.GithubUtility {
	client, err := githubAppClient.GetClientForInstallation(installationId)
	if err != nil {
		return nil
	}
	return &GitHubUtility{client}
}

func (g *GitHubAppClient) GetClientForInstallation(installationID int64) (*github.Client, error) {
	itr, err := ghinstallation.NewKeyFromFile(
		http.DefaultTransport,
		g.appID,
		installationID,
		g.privateKeyPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create installation transport: %w", err)
	}

	return github.NewClient(&http.Client{Transport: itr}), nil
}
