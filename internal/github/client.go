package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Masterminds/semver/v3"
	"github.com/adamkirk/orca/internal/common"
	"github.com/google/go-github/v74/github"
)

type GithubClient struct {
	client *github.Client
}

func (c *GithubClient) LatestVersion(owner string, repo string) (*semver.Version, error) {
	rel, resp, err := c.client.Repositories.GetLatestRelease(context.TODO(), owner, repo)

	if err != nil {
		return nil, common.ErrUnexpectedApiError{
			Msg: fmt.Sprintf("error from client: %s", err.Error()),
		}
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return semver.NewVersion(*rel.Name)
	case http.StatusTooManyRequests:
		return nil, common.ErrRateLimitedByApi{
			Api: "github",
		}
	default:
		return nil, common.ErrUnexpectedApiError{
			Msg: fmt.Sprintf("got %d response when getting release", resp.StatusCode),
		}
	}
}

func (c *GithubClient) VersionExists(owner string, repo string, v string) (bool, error) {
	_, resp, err := c.client.Repositories.GetReleaseByTag(context.TODO(), owner, repo, v)

	if err != nil {
		return false, common.ErrUnexpectedApiError{
			Msg: fmt.Sprintf("error from client: %s", err.Error()),
		}
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusTooManyRequests:
		return false, common.ErrRateLimitedByApi{
			Api: "github",
		}
	default:
		return false, common.ErrUnexpectedApiError{
			Msg: fmt.Sprintf("got %d response when getting release", resp.StatusCode),
		}
	}
}

// GetFileContents fetches the contents of the file at the given path on the
// repository's default branch. The bool is false if the file does not exist.
func (c *GithubClient) GetFileContents(owner string, repo string, path string) ([]byte, bool, error) {
	file, _, resp, err := c.client.Repositories.GetContents(context.TODO(), owner, repo, path, nil)

	var rateLimitErr *github.RateLimitError
	var abuseRateLimitErr *github.AbuseRateLimitError

	if errors.As(err, &rateLimitErr) || errors.As(err, &abuseRateLimitErr) {
		return nil, false, common.ErrRateLimitedByApi{
			Api: "github",
		}
	}

	if resp == nil {
		return nil, false, common.ErrUnexpectedApiError{
			Msg: fmt.Sprintf("error from client: %s", err.Error()),
		}
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// A directory at this path comes back without a file
		if file == nil {
			return nil, false, nil
		}

		contents, err := file.GetContent()

		if err != nil {
			return nil, false, common.ErrUnexpectedApiError{
				Msg: fmt.Sprintf("failed to decode file contents: %s", err.Error()),
			}
		}

		return []byte(contents), true, nil
	case http.StatusNotFound:
		return nil, false, nil
	case http.StatusTooManyRequests:
		return nil, false, common.ErrRateLimitedByApi{
			Api: "github",
		}
	default:
		return nil, false, common.ErrUnexpectedApiError{
			Msg: fmt.Sprintf("got %d response when getting file contents", resp.StatusCode),
		}
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
