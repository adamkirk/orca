package workspaces

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
	"github.com/spf13/afero"
)

type InitialiseCommand struct {
	bus.Returns[bus.Void]

	// A github repository in the format 'owner/repo' to clone the workspace
	// config from, takes precedence over SourceDirectory
	SourceRepo string

	SourceDirectory   string
	WorkspaceFileName string
	Into              string
}

func (InitialiseCommand) MessageName() string {
	return "ws.init"
}

func (m *Manager) Initialise(ctx context.Context, cmd InitialiseCommand) error {
	if cmd.SourceRepo != "" {
		return m.InitialiseFromRemoteRepo(ctx, cmd.SourceRepo, cmd.WorkspaceFileName, cmd.Into)
	}

	return m.InitialiseFromLocalDirectory(ctx, cmd.SourceDirectory, cmd.WorkspaceFileName, cmd.Into)
}

func (m *Manager) InitialiseFromLocalDirectory(ctx context.Context, dir string, workspaceFile string, into string) error {
	workspaceConfigPath := fmt.Sprintf("%s/%s", dir, workspaceFile)
	exists, err := afero.Exists(m.fs, workspaceConfigPath)

	if err != nil {
		return m.tui.RecordIfError("Failed to find config, this is likely a bug!", err)
	}

	if !exists {
		return m.tui.RecordIfError(
			fmt.Sprintf("Could not find workspace config: %s", workspaceConfigPath),
			common.ErrFileNotFound{
				Path: workspaceConfigPath,
			},
		)
	}

	cfg, err := m.workspaceRepo.LoadUnconfiguredWorkspace(workspaceConfigPath)

	if err != nil {
		return m.tui.RecordIfError(fmt.Sprintf("Failed to load workspace config '%s'!", workspaceConfigPath), err)
	}

	err = m.configManager.AddWorkspace(workspaceConfigPath, cfg.Name)

	if err != nil {
		if _, ok := err.(common.ErrWorkspaceAlreadyExists); ok {
			m.tui.Error("Workspace already exists!")
			return err
		}

		return m.tui.RecordIfError("Failed to register workspace, this is likely a bug!", err)
	}

	if err := m.clone(cfg.Name, "", into); err != nil {
		return err
	}

	m.tui.Success("Workspace initialised!")
	return nil
}

type githubRepo struct {
	Owner    string
	Name     string
	CloneURL string
}

var githubRepoPattern = regexp.MustCompile(`^[A-Za-z0-9-]+/[A-Za-z0-9._-]+$`)

// parseGithubRepo parses the 'owner/repo' format, the repo will be cloned over
// ssh.
func parseGithubRepo(input string) (githubRepo, error) {
	if !githubRepoPattern.MatchString(input) {
		return githubRepo{}, common.ErrInvalidInput{
			To:  "repo",
			Msg: fmt.Sprintf("'%s' is not in the format 'owner/repo'", input),
		}
	}

	parts := strings.Split(input, "/")

	return githubRepo{
		Owner:    parts[0],
		Name:     parts[1],
		CloneURL: fmt.Sprintf("git@github.com:%s/%s.git", parts[0], parts[1]),
	}, nil
}

func (m *Manager) InitialiseFromRemoteRepo(ctx context.Context, repoName string, workspaceFile string, into string) error {
	repo, err := parseGithubRepo(repoName)

	if err != nil {
		return m.tui.RecordIfError(fmt.Sprintf("Invalid repository '%s', expected the format 'owner/repo'!", repoName), err)
	}

	workspaceFilePath := strings.TrimPrefix(path.Clean(workspaceFile), "/")

	m.tui.Info(fmt.Sprintf("Checking for '%s' in %s/%s...", workspaceFilePath, repo.Owner, repo.Name))

	contents, exists, err := m.githubClient.GetFileContents(repo.Owner, repo.Name, workspaceFilePath)

	if err != nil {
		return m.tui.RecordIfError("Failed to check repository for workspace config!", err)
	}

	if !exists {
		return m.tui.RecordIfError(
			fmt.Sprintf(
				"Could not find workspace config '%s' in %s/%s (if the repository is private, set GH_TOKEN or GITHUB_TOKEN)",
				workspaceFilePath,
				repo.Owner,
				repo.Name,
			),
			common.ErrFileNotFound{
				Path: workspaceFilePath,
			},
		)
	}

	if _, err := m.workspaceRepo.ParseUnconfiguredWorkspace(contents, workspaceFilePath); err != nil {
		return m.tui.RecordIfError(
			fmt.Sprintf("Workspace config '%s' in %s/%s is invalid!", workspaceFilePath, repo.Owner, repo.Name),
			err,
		)
	}

	cloneDir, err := filepath.Abs(filepath.Join(into, repo.Name))

	if err != nil {
		return m.tui.RecordIfError("Failed to determine clone directory!", err)
	}

	dirExists, err := afero.Exists(m.fs, cloneDir)

	if err != nil {
		return err
	}

	if dirExists {
		return m.tui.RecordIfError(
			fmt.Sprintf("Cannot clone into '%s', it already exists!", cloneDir),
			common.ErrDirectoryAlreadyExists{
				Path: cloneDir,
			},
		)
	}

	if err := m.fs.MkdirAll(into, 0755); err != nil {
		return m.tui.RecordIfError("Failed to create target directory!", err)
	}

	m.tui.Info(fmt.Sprintf("Cloning '%s' into %s...", repo.CloneURL, cloneDir))

	if err := m.git.Clone(repo.CloneURL, cloneDir, ""); err != nil {
		return m.tui.RecordIfError("Clone failed!", err)
	}

	return m.InitialiseFromLocalDirectory(ctx, cloneDir, workspaceFile, into)
}
