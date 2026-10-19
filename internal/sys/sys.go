package sys

import (
	"github.com/Masterminds/semver/v3"
)

const orcaRepoOrg = "adamkirk"
const orcaRepoName = "orca"

// const orcaRepo = "adamkirk/orca"

type tui interface {
	Info(msg ...string)
	Error(msg ...string)
	Success(msg ...string)
}

type githubClient interface {
	VersionExists(owner string, repo string, v string) (bool, error)
	LatestVersion(owner string, repo string) (*semver.Version, error)
}

type Manager struct {
	tui    tui
	github githubClient
}

func NewManager(tui tui, github githubClient) *Manager {
	return &Manager{
		tui:    tui,
		github: github,
	}
}
