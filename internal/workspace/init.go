package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/spf13/afero"
)

type InitialiseCommand struct {
	bus.Returns[bus.Void]

	// A github repository in the format 'owner/repo' to clone the workspace
	// config from, takes precedence over SourceDirectory
	SourceRepo string

	// The branch of SourceRepo to read the workspace config from, and to
	// clone. Empty means the repository's default branch.
	Branch string

	SourceDirectory   string
	WorkspaceFileName string
	Into              string
}

func (InitialiseCommand) MessageName() string {
	return "ws.init"
}

func (m *Manager) Initialise(ctx context.Context, cmd InitialiseCommand) error {
	if cmd.SourceRepo != "" {
		return m.InitialiseFromRemoteRepo(ctx, cmd.SourceRepo, cmd.Branch, cmd.WorkspaceFileName, cmd.Into)
	}

	if cmd.Branch != "" {
		return usererr.Wrap(
			fmt.Errorf("%w to 'ws.init': 'branch' can only be used with 'repo'", ErrInvalidInput),
			"A branch can only be given when initialising from a repository (--repo)!",
		)
	}

	return m.InitialiseFromLocalDirectory(ctx, cmd.SourceDirectory, cmd.WorkspaceFileName, cmd.Into, false)
}

// InitialiseFromLocalDirectory registers the workspace config in dir, and
// clones its projects. branchesChecked skips checking the projects' branches
// exist, when the caller has already done so.
//
// The branches are checked before the workspace is registered, so a missing
// branch leaves nothing behind.
func (m *Manager) InitialiseFromLocalDirectory(ctx context.Context, dir string, workspaceFile string, into string, branchesChecked bool) error {
	// The path is registered, and every command loads the config from it,
	// wherever it's run, so it has to be absolute
	dir, err := filepath.Abs(dir)

	if err != nil {
		return usererr.Wrap(err, "Failed to determine the workspace's directory!")
	}

	workspaceConfigPath := filepath.Join(dir, workspaceFile)
	exists, err := afero.Exists(m.fs, workspaceConfigPath)

	if err != nil {
		return usererr.Wrap(err, "Failed to find config, this is likely a bug!")
	}

	if !exists {
		return usererr.Wrap(
			fmt.Errorf("%w: %s", ErrFileNotFound, workspaceConfigPath),
			fmt.Sprintf("Could not find workspace config: %s", workspaceConfigPath),
		)
	}

	cfg, err := m.workspaceRepo.LoadUnconfiguredWorkspace(workspaceConfigPath)

	if err != nil {
		return usererr.Wrap(err, fmt.Sprintf("Failed to load workspace config '%s'!", workspaceConfigPath))
	}

	if !branchesChecked {
		if err := m.checkWorkspaceBranches(cfg, into); err != nil {
			return err
		}
	}

	err = m.configManager.AddWorkspace(workspaceConfigPath, cfg.Name)

	if err != nil {
		if errors.Is(err, ErrWorkspaceAlreadyExists) {
			return usererr.Wrap(err, "Workspace already exists!")
		}

		return usererr.Wrap(err, "Failed to register workspace, this is likely a bug!")
	}

	if err := m.clone(cfg.Name, "", into, true); err != nil {
		return err
	}

	if err := m.createPreferences(cfg); err != nil {
		return err
	}

	m.tui.Success("Workspace initialised!")
	m.suggestIgnoringPreferences(cfg.PreferencesPath)

	return nil
}

// createPreferences creates the workspace's preferences file, listing every
// project, unless it already exists.
func (m *Manager) createPreferences(cfg *UnconfiguredWorkspace) error {
	names := make([]string, len(cfg.Projects))

	for i, p := range cfg.Projects {
		names[i] = p.Name
	}

	created, err := writeDefaultPreferences(m.fs, cfg.PreferencesPath, names)

	if err != nil {
		return usererr.Wrap(err, fmt.Sprintf("Failed to create the preferences file '%s'!", cfg.PreferencesPath))
	}

	if created {
		m.tui.Info(fmt.Sprintf("Created %s, where you can enable and disable projects.", cfg.PreferencesPath))
	}

	return nil
}

// suggestIgnoringPreferences suggests gitignoring the preferences file, which
// is usually personal, if it isn't already.
func (m *Manager) suggestIgnoringPreferences(path string) {
	ignored, err := m.git.IsIgnored(path)

	if err != nil {
		slog.Debug("couldn't check whether the preferences file is ignored", "err", err)
		return
	}

	if ignored {
		return
	}

	m.tui.Info(
		"",
		"Next steps:",
		fmt.Sprintf("  %s holds your own preferences, like which projects are enabled.", path),
		"  It isn't ignored by git, so unless your team commits it as shared defaults, add it to the workspace repository's .gitignore.",
	)
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
		return githubRepo{}, fmt.Errorf("%w to 'repo': '%s' is not in the format 'owner/repo'", ErrInvalidInput, input)
	}

	parts := strings.Split(input, "/")

	return githubRepo{
		Owner:    parts[0],
		Name:     parts[1],
		CloneURL: fmt.Sprintf("git@github.com:%s/%s.git", parts[0], parts[1]),
	}, nil
}

func (m *Manager) InitialiseFromRemoteRepo(ctx context.Context, repoName string, branch string, workspaceFile string, into string) error {
	repo, err := parseGithubRepo(repoName)

	if err != nil {
		return usererr.Wrap(err, fmt.Sprintf("Invalid repository '%s', expected the format 'owner/repo'!", repoName))
	}

	if branch != "" {
		exists, err := m.git.RemoteBranchExists(repo.CloneURL, branch)

		if err != nil {
			return usererr.Wrap(err, fmt.Sprintf("Failed to check branch '%s' of %s/%s!", branch, repo.Owner, repo.Name))
		}

		if !exists {
			return usererr.Wrap(
				fmt.Errorf("%w: %s/%s: '%s'", ErrBranchNotFound, repo.Owner, repo.Name, branch),
				fmt.Sprintf("Could not find branch '%s' in %s/%s!", branch, repo.Owner, repo.Name),
			)
		}
	}

	workspaceFilePath := strings.TrimPrefix(path.Clean(workspaceFile), "/")

	m.tui.Info(fmt.Sprintf("Checking for '%s' in %s/%s%s...", workspaceFilePath, repo.Owner, repo.Name, branchSuffix(branch)))

	contents, exists, err := m.githubClient.GetFileContents(repo.Owner, repo.Name, workspaceFilePath, branch)

	if err != nil {
		return usererr.Wrap(err, "Failed to check repository for workspace config!")
	}

	if !exists {
		return usererr.Wrap(
			fmt.Errorf("%w: %s", ErrFileNotFound, workspaceFilePath),
			fmt.Sprintf(
				"Could not find workspace config '%s' in %s/%s (if the repository is private, set GH_TOKEN or GITHUB_TOKEN)",
				workspaceFilePath,
				repo.Owner,
				repo.Name,
			),
		)
	}

	wsCfg, err := m.workspaceRepo.ParseUnconfiguredWorkspace(contents, workspaceFilePath)

	if err != nil {
		return usererr.Wrap(
			err,
			fmt.Sprintf("Workspace config '%s' in %s/%s is invalid!", workspaceFilePath, repo.Owner, repo.Name),
		)
	}

	cloneDir, err := filepath.Abs(filepath.Join(into, repo.Name))

	if err != nil {
		return usererr.Wrap(err, "Failed to determine clone directory!")
	}

	dirExists, err := afero.Exists(m.fs, cloneDir)

	if err != nil {
		return err
	}

	if dirExists {
		return usererr.Wrap(
			fmt.Errorf("%w: %s", ErrDirectoryAlreadyExists, cloneDir),
			fmt.Sprintf("Cannot clone into '%s', it already exists!", cloneDir),
		)
	}

	// Check the projects' branches before cloning anything, including the
	// workspace repo itself
	if err := m.checkWorkspaceBranches(wsCfg, into); err != nil {
		return err
	}

	if err := m.fs.MkdirAll(into, 0755); err != nil {
		return usererr.Wrap(err, "Failed to create target directory!")
	}

	m.tui.Info(fmt.Sprintf("Cloning '%s'%s into %s...", repo.CloneURL, branchSuffix(branch), cloneDir))

	if err := m.git.Clone(repo.CloneURL, cloneDir, branch); err != nil {
		return usererr.Wrap(err, "Clone failed!")
	}

	return m.InitialiseFromLocalDirectory(ctx, cloneDir, workspaceFile, into, true)
}

// checkWorkspaceBranches checks the branches of a workspace's projects before
// it's registered, as clone would for the projects cloned into the target.
func (m *Manager) checkWorkspaceBranches(cfg *UnconfiguredWorkspace, into string) error {
	targetDir, err := filepath.Abs(into)

	if err != nil {
		return usererr.Wrap(err, "Failed to determine target directory!")
	}

	targets := make([]cloneTarget, len(cfg.Projects))

	for i, p := range cfg.Projects {
		project := Project{Name: p.Name, Type: p.Type, RepositoryConfig: p.RepositoryConfig, Path: p.Path}

		targets[i] = cloneTarget{
			project: project,
			into:    cloneDir(targetDir, project),
		}
	}

	return m.checkBranches(targets)
}
