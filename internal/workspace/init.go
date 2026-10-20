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

	// Clone without running any hooks
	SkipHooks bool

	// Run every extension the hooks run without asking, and without recording
	// them as trusted
	Yes bool
}

// initOpts are the options for initialising a workspace, from a directory or a
// repository.
type initOpts struct {
	workspaceFile string
	into          string

	// Skip checking the projects' branches exist, when the caller has already
	branchesChecked bool

	skipHooks bool
	yes       bool
}

func (InitialiseCommand) MessageName() string {
	return "ws.init"
}

func (m *Manager) Initialise(ctx context.Context, cmd InitialiseCommand) error {
	opts := initOpts{
		workspaceFile: cmd.WorkspaceFileName,
		into:          cmd.Into,
		skipHooks:     cmd.SkipHooks,
		yes:           cmd.Yes,
	}

	if cmd.SourceRepo != "" {
		return m.initialiseFromRemoteRepo(cmd.SourceRepo, cmd.Branch, opts)
	}

	if cmd.Branch != "" {
		return usererr.Wrap(
			fmt.Errorf("%w to 'ws.init': 'branch' can only be used with 'repo'", ErrInvalidInput),
			"A branch can only be given when initialising from a repository (--repo)!",
		)
	}

	return m.initialiseFromLocalDirectory(cmd.SourceDirectory, opts)
}

// initialiseFromLocalDirectory registers the workspace config in dir, creates
// the preferences file, clones its projects, and runs their hooks. It's safe to
// run again: a workspace that's already registered from the same config carries
// on, cloning whatever's missing, and running the hooks again.
//
// The branches are checked before the workspace is registered, so a missing
// branch leaves nothing behind.
func (m *Manager) initialiseFromLocalDirectory(dir string, opts initOpts) error {
	// The path is registered, and every command loads the config from it,
	// wherever it's run, so it has to be absolute
	dir, err := filepath.Abs(dir)

	if err != nil {
		return usererr.Wrap(err, "Failed to determine the workspace's directory!")
	}

	workspaceConfigPath := filepath.Join(dir, opts.workspaceFile)
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

	if !opts.branchesChecked {
		if err := m.checkWorkspaceBranches(cfg, opts.into); err != nil {
			return err
		}
	}

	if err := m.registerWorkspace(cfg.Name, workspaceConfigPath); err != nil {
		return err
	}

	if err := m.createPreferences(cfg); err != nil {
		return err
	}

	ws, err := m.clone(cfg.Name, cloneOpts{
		to:              opts.into,
		branchesChecked: true,
		skipHooks:       opts.skipHooks,
		yes:             opts.yes,
		initialise:      true,
	})

	if err != nil {
		return err
	}

	m.showNextSteps(ws)
	m.suggestIgnoringPreferences(cfg.PreferencesPath)

	return nil
}

// registerWorkspace records the workspace in the user config. One that's
// already registered from the same config is left as it is, so init can be run
// again, but another workspace with the same name is an error.
func (m *Manager) registerWorkspace(name string, configPath string) error {
	err := m.configManager.AddWorkspace(configPath, name)

	if !errors.Is(err, ErrWorkspaceAlreadyExists) {
		return usererr.Wrap(err, "Failed to register workspace, this is likely a bug!")
	}

	existing, metaErr := m.configManager.GetWorkspaceMeta(name)

	if metaErr != nil {
		return usererr.Wrap(metaErr, "Failed to check the existing workspace, this is likely a bug!")
	}

	if filepath.Clean(existing.Path) != filepath.Clean(configPath) {
		return usererr.Wrap(err, fmt.Sprintf("There's already a workspace called %s, from %s!", name, existing.Path))
	}

	m.tui.Info(fmt.Sprintf("Workspace %s is already initialised here; cloning anything that's missing, and running its hooks again.", name))

	return nil
}

// showNextSteps lists what's left to do to run the workspace, which orca never
// does itself, e.g. as it needs sudo.
func (m *Manager) showNextSteps(ws *Workspace) {
	steps := [][2]string{}

	if m.configManager.GetCurrentWorkspace() != ws.Name {
		steps = append(steps, [2]string{fmt.Sprintf("orca ws switch %s", ws.Name), "make it your current workspace"})
	}

	if len(ws.GetUniqueHosts()) > 0 {
		steps = append(steps, [2]string{`sudo ORCA_CONFIG_PATH="$HOME/.orca/orca.yaml" orca hosts`, "add its hosts to /etc/hosts"})
	}

	if len(ws.GetUniqueTLSCertificates()) > 0 {
		steps = append(steps, [2]string{"orca tls gen", "generate its TLS certificates"})
	}

	steps = append(steps, [2]string{"orca up", "start it"})

	width := 0

	for _, step := range steps {
		width = max(width, len(step[0]))
	}

	lines := []string{}

	for _, step := range steps {
		lines = append(lines, fmt.Sprintf("  %-*s   %s", width, step[0], step[1]))
	}

	m.tui.Info("")
	m.tui.Success(fmt.Sprintf("Workspace %s is ready. Next steps:", ws.Name))
	m.tui.Info(lines...)
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
		fmt.Sprintf("Note: %s holds your own preferences, like which projects are enabled.", path),
		"It isn't ignored by git, so unless your team commits it as shared defaults, add it to the workspace repository's .gitignore.",
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

// initialiseFromRemoteRepo clones the workspace's repository into opts.into,
// unless it's already there, then initialises it from there.
func (m *Manager) initialiseFromRemoteRepo(repoName string, branch string, opts initOpts) error {
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

	workspaceFilePath := strings.TrimPrefix(path.Clean(opts.workspaceFile), "/")

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

	cloneDir, err := filepath.Abs(filepath.Join(opts.into, repo.Name))

	if err != nil {
		return usererr.Wrap(err, "Failed to determine clone directory!")
	}

	alreadyCloned, err := m.isClonedAt(cloneDir, repo.CloneURL)

	if err != nil {
		return err
	}

	// Check the projects' branches before cloning anything, including the
	// workspace repo itself
	if err := m.checkWorkspaceBranches(wsCfg, opts.into); err != nil {
		return err
	}

	if alreadyCloned {
		m.tui.Info(fmt.Sprintf("%s/%s is already cloned at %s, so it's used as it is.", repo.Owner, repo.Name, cloneDir))
	} else {
		if err := m.fs.MkdirAll(opts.into, 0755); err != nil {
			return usererr.Wrap(err, "Failed to create target directory!")
		}

		m.tui.Info(fmt.Sprintf("Cloning '%s'%s into %s...", repo.CloneURL, branchSuffix(branch), cloneDir))

		if err := m.git.Clone(repo.CloneURL, cloneDir, branch); err != nil {
			return usererr.Wrap(err, "Clone failed!")
		}
	}

	opts.branchesChecked = true

	return m.initialiseFromLocalDirectory(cloneDir, opts)
}

// isClonedAt reports whether dir is already a clone of the repo, e.g. as init
// is being run again. Anything else already at dir is an error, as the repo
// can't be cloned there.
func (m *Manager) isClonedAt(dir string, repoURL string) (bool, error) {
	exists, err := afero.Exists(m.fs, dir)

	if err != nil || !exists {
		return false, err
	}

	hasOrigin, err := m.git.DirectoryHasOrigin(dir, repoURL)

	if err != nil {
		return false, usererr.Wrap(err, fmt.Sprintf("Failed to check what's already in '%s'!", dir))
	}

	if !hasOrigin {
		return false, usererr.Wrap(
			fmt.Errorf("%w: %s", ErrDirectoryAlreadyExists, dir),
			fmt.Sprintf("Cannot clone into '%s', it already exists, and isn't a clone of %s!", dir, repoURL),
		)
	}

	return true, nil
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
