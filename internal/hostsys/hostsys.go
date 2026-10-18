package hostsys

import (
	"github.com/Masterminds/semver/v3"
	"github.com/spf13/afero"
)

const orcaRepoOrg = "adamkirk"
const orcaRepoName = "orca"

// const orcaRepo = "adamkirk/orca"

type tui interface {
	Info(msg ...string)
	Error(msg ...string)
	Success(msg ...string)
	RecordIfError(msg string, err error) error
}

type githubClient interface {
	VersionExists(owner string, repo string, v string) (bool, error)
	LatestVersion(owner string, repo string) (*semver.Version, error)
}

type HostSystem struct {
	tui      tui
	fs       afero.Fs
	toolsDir string
	github   githubClient
}

func NewHostSystem(tui tui, fs afero.Fs, github githubClient, toolsDir string) *HostSystem {
	return &HostSystem{
		tui:      tui,
		fs:       fs,
		github:   github,
		toolsDir: toolsDir,
	}
}
