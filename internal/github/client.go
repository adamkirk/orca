package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v74/github"
)

type GithubClient struct {
	client *github.Client
}

func (c *GithubClient) LatestVersion(owner string, repo string) (*semver.Version, error) {
	rel, resp, err := c.client.Repositories.GetLatestRelease(context.TODO(), owner, repo)

	if err != nil {
		return nil, fmt.Errorf("%w: error from client: %w", ErrUnexpectedAPIResponse, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return semver.NewVersion(*rel.Name)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w for 'github'", ErrRateLimited)
	default:
		return nil, fmt.Errorf("%w: got %d response when getting release", ErrUnexpectedAPIResponse, resp.StatusCode)
	}
}

func (c *GithubClient) VersionExists(owner string, repo string, v string) (bool, error) {
	_, resp, err := c.client.Repositories.GetReleaseByTag(context.TODO(), owner, repo, v)

	if err != nil {
		return false, fmt.Errorf("%w: error from client: %w", ErrUnexpectedAPIResponse, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusTooManyRequests:
		return false, fmt.Errorf("%w for 'github'", ErrRateLimited)
	default:
		return false, fmt.Errorf("%w: got %d response when getting release", ErrUnexpectedAPIResponse, resp.StatusCode)
	}
}

// GetFileContents fetches the contents of the file at the given path on the
// given ref, or the repository's default branch when it's empty. The bool is
// false if the file does not exist.
func (c *GithubClient) GetFileContents(owner string, repo string, path string, ref string) ([]byte, bool, error) {
	file, _, resp, err := c.client.Repositories.GetContents(
		context.TODO(),
		owner,
		repo,
		path,
		&github.RepositoryContentGetOptions{Ref: ref},
	)

	var rateLimitErr *github.RateLimitError
	var abuseRateLimitErr *github.AbuseRateLimitError

	if errors.As(err, &rateLimitErr) || errors.As(err, &abuseRateLimitErr) {
		return nil, false, fmt.Errorf("%w for 'github'", ErrRateLimited)
	}

	if resp == nil {
		return nil, false, fmt.Errorf("%w: error from client: %w", ErrUnexpectedAPIResponse, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// A directory at this path comes back without a file
		if file == nil {
			return nil, false, nil
		}

		contents, err := file.GetContent()

		if err != nil {
			return nil, false, fmt.Errorf("%w: failed to decode file contents: %w", ErrUnexpectedAPIResponse, err)
		}

		return []byte(contents), true, nil
	case http.StatusNotFound:
		return nil, false, nil
	case http.StatusTooManyRequests:
		return nil, false, fmt.Errorf("%w for 'github'", ErrRateLimited)
	default:
		return nil, false, fmt.Errorf("%w: got %d response when getting file contents", ErrUnexpectedAPIResponse, resp.StatusCode)
	}
}

// NewGithubClient creates a client for the github API, if the token is empty
// then requests are unauthenticated.
func NewGithubClient(token string) *GithubClient {
	client := github.NewClient(nil)

	if token != "" {
		client = client.WithAuthToken(token)
	}

	return &GithubClient{
		client: client,
	}
}
