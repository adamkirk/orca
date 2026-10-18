package workspaces

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
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

func (m *Manager) setupAllProjects(ws *common.Workspace, into string) error {
	for _, project := range ws.Projects {
		dir := fmt.Sprintf("%s/%s", into, project.Name)

		if err := m.cloneSingleProject(ws, project, dir); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) registerWorkspaceProject(ws *common.Workspace, project common.Project) error {
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

func (m *Manager) cloneSingleProject(ws *common.Workspace, project common.Project, into string) error {
	projectExists, err := m.configManager.ProjectExists(ws.Name, project.Name)

	if err != nil {
		return err
	}

	if project.RepositoryConfig.Self {
		m.tui.Info(fmt.Sprintf("Registering workspace as project: %s", project.Name))

		return m.tui.RecordIfError(
			"Failed to register workspace as a project!",
			m.registerWorkspaceProject(ws, project),
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
			return m.tui.RecordIfError(
				"Failed to verify existing directory!",
				err,
			)
		}

		if isCloned {
			m.tui.Info(fmt.Sprintf("Skipping '%s' already cloned at '%s'!", project.Name, into))

			return m.tui.RecordIfError(
				"Failed to save config, you will have to edit this manually but adding the path into your orca config!",
				m.configManager.SetProjectPath(ws.Name, project.Name, into),
			)
		}

		return common.ErrDirectoryDoesNotMatchOrigin{
			Path:   into,
			Origin: project.RepositoryConfig.SSH,
		}
	}

	parentDir := filepath.Dir(into)

	if err := m.fs.MkdirAll(parentDir, 0755); err != nil {
		return err
	}

	m.tui.Info(fmt.Sprintf("Cloning '%s' into %s...", project.RepositoryConfig.SSH, into))

	err = m.git.Clone(project.RepositoryConfig.SSH, into, "")

	if err != nil {
		return m.tui.RecordIfError("Clone failed!", err)
	}

	return m.tui.RecordIfError(
		"Failed to save config, you will have to edit this manually but adding the path into your orca config!",
		m.configManager.SetProjectPath(ws.Name, project.Name, into),
	)
}

func (m *Manager) projectIsAlreadyClonedAtLocation(proj common.Project, into string) (bool, error) {
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

func (m *Manager) Clone(_ context.Context, cmd CloneCommand) error {
	ctx, err := m.contextResolver.Resolve(cmd.WorkspaceName, "")

	if err != nil {
		return m.tui.RecordIfError("could not resolve execution context", err)
	}

	wsMeta, err := m.configManager.GetWorkspaceMeta(ctx.Workspace.Name)

	if err != nil {
		if _, ok := err.(common.ErrUnknownWorkspace); ok {
			return m.tui.RecordIfError(fmt.Sprintf("Unknown workspace: %s", cmd.WorkspaceName), err)
		}

		return err
	}

	cfg, err := m.workspaceRepo.Load(wsMeta.Name)

	if err != nil {
		return err
	}

	targetDir, err := m.getCloneTargetDir(wsMeta.Path, cmd.To)

	if err != nil {
		return m.tui.RecordIfError("Failed to determine target directory!", err)
	}

	var cloneErr error = nil

	if cmd.Project != "" {
		project, err := cfg.GetProject(cmd.Project)

		if err != nil {
			return m.tui.RecordIfError("Failed to clone specified project!", err)
		}

		targetDir := targetDir

		if cmd.To == "" {
			targetDir = fmt.Sprintf("%s/%s", targetDir, project.Name)
		}

		cloneErr = m.cloneSingleProject(cfg, *project, targetDir)
	} else {
		cloneErr = m.setupAllProjects(cfg, targetDir)
	}

	if cloneErr != nil {
		if _, ok := cloneErr.(common.ErrDirectoryAlreadyExists); ok {
			return m.tui.RecordIfError("Cannot clone into the given directory, it already exists!", cloneErr)
		}

		return cloneErr
	}

	return nil
}
