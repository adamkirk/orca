package workspace

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/spf13/afero"
)

type CloneCommand struct {
	bus.Returns[bus.Void]

	WorkspaceName string
	Project       string
	To            string
}

func (CloneCommand) MessageName() string {
	return "ws.clone"
}

// cloneTarget is a project and the directory it's cloned into.
type cloneTarget struct {
	project Project
	into    string
}

// cloneDir is the directory a project is cloned into, within targetDir: its
// path if it has one, otherwise its name.
func cloneDir(targetDir string, p Project) string {
	return filepath.Join(targetDir, cmp.Or(p.Path, p.Name))
}

// cloneOrder orders targets so each project is cloned after the projects it
// requires, so their hooks can rely on them, and after any project whose
// directory it's inside, which couldn't be cloned into a directory that
// already exists. Otherwise, the configured order is kept.
func cloneOrder(targets []cloneTarget) ([]cloneTarget, error) {
	after := map[string][]string{}

	for _, t := range targets {
		for _, other := range targets {
			if other.project.Name == t.project.Name {
				continue
			}

			if slices.Contains(t.project.Requires, other.project.Name) || encloses(other, t) {
				after[t.project.Name] = append(after[t.project.Name], other.project.Name)
			}
		}
	}

	ordered := make([]cloneTarget, 0, len(targets))
	done := map[string]bool{}

	for len(ordered) < len(targets) {
		i := slices.IndexFunc(targets, func(t cloneTarget) bool {
			return !done[t.project.Name] && !slices.ContainsFunc(after[t.project.Name], func(name string) bool {
				return !done[name]
			})
		})

		if i == -1 {
			remaining := []string{}

			for _, t := range targets {
				if !done[t.project.Name] {
					remaining = append(remaining, t.project.Name)
				}
			}

			return nil, fmt.Errorf("%w: %s", ErrCircularCloneOrder, strings.Join(remaining, ", "))
		}

		ordered = append(ordered, targets[i])
		done[targets[i].project.Name] = true
	}

	return ordered, nil
}

// encloses reports whether t is cloned somewhere inside outer's directory. The
// workspace's own repository isn't cloned, so it never encloses anything.
func encloses(outer cloneTarget, t cloneTarget) bool {
	if outer.project.RepositoryConfig.Self {
		return false
	}

	inner, dir := filepath.Clean(t.into), filepath.Clean(outer.into)

	return inner != dir && isWithinDir(inner, dir)
}

// enclosing returns the project whose repository t is cloned inside, the
// deepest one if there are several, or nil.
func enclosing(t cloneTarget, targets []cloneTarget) *cloneTarget {
	var outer *cloneTarget

	for i := range targets {
		if encloses(targets[i], t) && (outer == nil || encloses(*outer, targets[i])) {
			outer = &targets[i]
		}
	}

	return outer
}

// warnIfNotIgnored warns when t is cloned inside another project's repository,
// which doesn't ignore it, so it would show up there as untracked files.
func (m *Manager) warnIfNotIgnored(t cloneTarget, targets []cloneTarget) {
	outer := enclosing(t, targets)

	if outer == nil {
		return
	}

	ignored, err := m.git.IsIgnored(t.into)

	if err != nil {
		slog.Debug("couldn't check whether a nested project is ignored", "project", t.project.Name, "err", err)
		return
	}

	if ignored {
		return
	}

	rel, err := filepath.Rel(outer.into, t.into)

	if err != nil {
		rel = t.into
	}

	m.tui.Info(fmt.Sprintf(
		"Warning: %s is inside %s, whose repository doesn't ignore %s, so it shows up there as untracked files. Add it to %s's .gitignore.",
		t.project.Name, outer.project.Name, rel, outer.project.Name,
	))
}

// checkEnclosingCloned fails if a project would be cloned inside another that
// isn't cloned, and isn't being cloned first. Cloning it would create the
// other's directory, which then couldn't be cloned into.
func (m *Manager) checkEnclosingCloned(targets []cloneTarget, all []cloneTarget) error {
	for _, t := range targets {
		outer := enclosing(t, all)

		if outer == nil || slices.ContainsFunc(targets, func(o cloneTarget) bool { return o.project.Name == outer.project.Name }) {
			continue
		}

		exists, err := afero.DirExists(m.fs, outer.into)

		if err != nil {
			return err
		}

		if !exists {
			return usererr.Wrap(
				fmt.Errorf("%w: %s is inside %s", ErrEnclosingProjectNotCloned, t.project.Name, outer.project.Name),
				fmt.Sprintf("%s is cloned inside %s, which isn't cloned yet; clone it first with 'orca ws clone -p %s'.", t.project.Name, outer.project.Name, outer.project.Name),
			)
		}
	}

	return nil
}

// register records where the project is, and its type, in the user config.
func (m *Manager) register(ws *Workspace, project Project, path string) error {
	return m.configManager.RegisterProject(ws.Name, ProjectMeta{
		Name:          project.Name,
		WorkspaceName: ws.Name,
		Path:          path,
		Type:          project.Type,
	})
}

// checkBranches checks that the branch of every project that is about to be
// cloned exists on its remote, so a typo stops orca before anything is
// cloned. Projects that are already cloned are skipped, as their branch is
// never changed.
func (m *Manager) checkBranches(targets []cloneTarget) error {
	var missing []string

	for _, t := range targets {
		repo := t.project.RepositoryConfig

		if repo.Self || repo.Branch == "" {
			continue
		}

		exists, err := afero.Exists(m.fs, t.into)

		if err != nil {
			return err
		}

		if exists {
			continue
		}

		branchExists, err := m.git.RemoteBranchExists(repo.SSH, repo.Branch)

		if err != nil {
			return usererr.Wrap(err, fmt.Sprintf("Failed to check branch '%s' of '%s'!", repo.Branch, repo.SSH))
		}

		if !branchExists {
			missing = append(missing, fmt.Sprintf("%s: '%s' (%s)", t.project.Name, repo.Branch, repo.SSH))
		}
	}

	if len(missing) == 0 {
		return nil
	}

	return usererr.Wrap(
		fmt.Errorf("%w: %s", ErrBranchNotFound, strings.Join(missing, ", ")),
		"Some branches don't exist on their remotes, nothing was cloned!",
	)
}

func (m *Manager) registerWorkspaceProject(ws *Workspace, project Project) error {
	wsLocation, err := m.configManager.GetWorkspaceMeta(ws.Name)

	if err != nil {
		return err
	}

	root, err := m.git.GetRepositoryRootFromPath(wsLocation.Path)

	if err != nil {
		return err
	}

	return m.register(ws, project, root)
}

func (m *Manager) cloneSingleProject(ws *Workspace, project Project, into string) error {
	projectExists, err := m.configManager.ProjectExists(ws.Name, project.Name)

	if err != nil {
		return err
	}

	if project.RepositoryConfig.Self {
		m.tui.Info(fmt.Sprintf("Registering workspace as project: %s", project.Name))

		return usererr.Wrap(
			m.registerWorkspaceProject(ws, project),
			"Failed to register workspace as a project!",
		)
	}

	exists, err := afero.Exists(m.fs, into)

	if err != nil {
		return err
	}

	if exists {
		if projectExists {
			m.tui.Info(fmt.Sprintf("Skipping '%s' as it already exists!", project.Name))
			return nil
		}

		isCloned, err := m.projectIsAlreadyClonedAtLocation(project, into)

		if err != nil {
			return usererr.Wrap(
				err,
				"Failed to verify existing directory!",
			)
		}

		if isCloned {
			m.tui.Info(fmt.Sprintf("Skipping '%s' already cloned at '%s'!", project.Name, into))

			return usererr.Wrap(
				m.register(ws, project, into),
				"Failed to save config, you will have to edit this manually but adding the path into your orca config!",
			)
		}

		return fmt.Errorf("%w: '%s' (expected %s)", ErrDirectoryDoesNotMatchOrigin, into, project.RepositoryConfig.SSH)
	}

	parentDir := filepath.Dir(into)

	if err := m.fs.MkdirAll(parentDir, 0755); err != nil {
		return err
	}

	m.tui.Info(fmt.Sprintf("Cloning '%s'%s%s into %s...", project.RepositoryConfig.SSH, typeSuffix(project), branchSuffix(project.RepositoryConfig.Branch), into))

	err = m.git.Clone(project.RepositoryConfig.SSH, into, project.RepositoryConfig.Branch)

	if err != nil {
		return usererr.Wrap(err, "Clone failed!")
	}

	return usererr.Wrap(
		m.register(ws, project, into),
		"Failed to save config, you will have to edit this manually but adding the path into your orca config!",
	)
}

func (m *Manager) projectIsAlreadyClonedAtLocation(proj Project, into string) (bool, error) {
	exists, err := afero.DirExists(m.fs, into)

	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	return m.git.DirectoryHasOrigin(into, proj.RepositoryConfig.SSH)
}

func (m *Manager) getCloneTargetDir(wsConfigPath string, target string) (string, error) {
	if target != "" {
		return filepath.Abs(target)
	}

	wsRoot, err := m.git.GetRepositoryRootFromPath(wsConfigPath)

	if err != nil {
		return "", err
	}

	return filepath.Abs(fmt.Sprintf("%s/..", wsRoot))
}

func (m *Manager) Clone(ctx context.Context, cmd CloneCommand) error {
	execCtx, err := ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return m.clone(execCtx.Workspace.Name, cmd.Project, cmd.To, false)
}

// branchSuffix describes the branch in progress messages, it's empty for the
// default branch.
func branchSuffix(branch string) string {
	if branch == "" {
		return ""
	}

	return fmt.Sprintf(" (branch '%s')", branch)
}

// typeSuffix labels libraries in progress messages, as they're never run.
func typeSuffix(project Project) string {
	if project.IsLibrary() {
		return " (library)"
	}

	return ""
}

// clone clones the given project, or all projects when it's empty, for the
// workspace into the given directory. branchesChecked skips checking the
// projects' branches exist, when the caller has already done so.
func (m *Manager) clone(wsName string, project string, to string, branchesChecked bool) error {
	wsMeta, err := m.configManager.GetWorkspaceMeta(wsName)

	if err != nil {
		if errors.Is(err, ErrUnknownWorkspace) {
			return usererr.Wrap(err, fmt.Sprintf("Unknown workspace: %s", wsName))
		}

		return err
	}

	cfg, err := m.workspaceRepo.Load(wsMeta.Name)

	if err != nil {
		return err
	}

	targetDir, err := m.getCloneTargetDir(wsMeta.Path, to)

	if err != nil {
		return usererr.Wrap(err, "Failed to determine target directory!")
	}

	var targets []cloneTarget

	if project != "" {
		project, err := cfg.GetProject(project)

		if err != nil {
			return usererr.Wrap(err, "Failed to clone specified project!")
		}

		into := targetDir

		if to == "" {
			into = cloneDir(targetDir, *project)
		}

		targets = append(targets, cloneTarget{project: *project, into: into})
	} else {
		for _, p := range cfg.Projects {
			targets = append(targets, cloneTarget{project: p, into: cloneDir(targetDir, p)})
		}

		targets, err = cloneOrder(targets)

		if err != nil {
			return usererr.Wrap(err, "Failed to work out the order to clone projects in!")
		}
	}

	if !branchesChecked {
		if err := m.checkBranches(targets); err != nil {
			return err
		}
	}

	// Other projects are found from all of them, so a project cloned on its
	// own is still checked against the one it's inside
	all := make([]cloneTarget, len(cfg.Projects))

	for i, p := range cfg.Projects {
		all[i] = cloneTarget{project: p, into: cloneDir(targetDir, p)}
	}

	if err := m.checkEnclosingCloned(targets, all); err != nil {
		return err
	}

	for _, t := range targets {
		if err := m.cloneSingleProject(cfg, t.project, t.into); err != nil {
			if errors.Is(err, ErrDirectoryAlreadyExists) {
				return usererr.Wrap(err, "Cannot clone into the given directory, it already exists!")
			}

			return err
		}

		if !t.project.RepositoryConfig.Self {
			m.warnIfNotIgnored(t, all)
		}
	}

	return nil
}
