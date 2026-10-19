package workspace

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
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

	return m.configManager.SetProjectPath(ws.Name, project.Name, root)
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
				m.configManager.SetProjectPath(ws.Name, project.Name, into),
				"Failed to save config, you will have to edit this manually but adding the path into your orca config!",
			)
		}

		return fmt.Errorf("%w: '%s' (expected %s)", ErrDirectoryDoesNotMatchOrigin, into, project.RepositoryConfig.SSH)
	}

	parentDir := filepath.Dir(into)

	if err := m.fs.MkdirAll(parentDir, 0755); err != nil {
		return err
	}

	m.tui.Info(fmt.Sprintf("Cloning '%s'%s into %s...", project.RepositoryConfig.SSH, branchSuffix(project.RepositoryConfig.Branch), into))

	err = m.git.Clone(project.RepositoryConfig.SSH, into, project.RepositoryConfig.Branch)

	if err != nil {
		return usererr.Wrap(err, "Clone failed!")
	}

	return usererr.Wrap(
		m.configManager.SetProjectPath(ws.Name, project.Name, into),
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

		targetDir := targetDir

		if to == "" {
			targetDir = fmt.Sprintf("%s/%s", targetDir, project.Name)
		}

		targets = append(targets, cloneTarget{project: *project, into: targetDir})
	} else {
		for _, p := range cfg.Projects {
			targets = append(targets, cloneTarget{project: p, into: fmt.Sprintf("%s/%s", targetDir, p.Name)})
		}
	}

	if !branchesChecked {
		if err := m.checkBranches(targets); err != nil {
			return err
		}
	}

	for _, t := range targets {
		if err := m.cloneSingleProject(cfg, t.project, t.into); err != nil {
			if errors.Is(err, ErrDirectoryAlreadyExists) {
				return usererr.Wrap(err, "Cannot clone into the given directory, it already exists!")
			}

			return err
		}
	}

	return nil
}
