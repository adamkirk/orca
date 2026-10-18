package provision

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/afero"
)

type tui interface {
	Success(msg ...string)
}

type Manager struct {
	fs  afero.Fs
	tui tui
}

func (m *Manager) RunAll(ws *workspace.Workspace, project *workspace.Project) error {
	for _, provisioner := range project.Config.Provisioners {
		if err := m.Run(ws, project, provisioner); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Run(ws *workspace.Workspace, project *workspace.Project, provisioner workspace.Provisioner) error {
	if provisioner.ExampleFile == nil {
		return fmt.Errorf("%w: the example file provisioner must be defined", ErrUnsupportedProvisioner)
	}

	srcPath := filepath.Join(project.ProjectDir, provisioner.ExampleFile.Src)
	targetPath := filepath.Join(project.ProjectDir, provisioner.ExampleFile.Target)
	targetExists, err := afero.Exists(m.fs, targetPath)

	if err != nil {
		return err
	}

	if targetExists {
		slog.Debug("target for 'example file' provisioner already exists", "path", targetPath, "project", project.Name, "workspace", ws.Name)
		return nil
	}

	srcExists, err := afero.Exists(m.fs, srcPath)

	if err != nil {
		return err
	}

	if !srcExists {
		return usererr.Wrap(
			fmt.Errorf("%w: %s", workspace.ErrFileNotFound, srcPath),
			"Failed to run provisioner, source file not found",
		)
	}

	srcContents, err := afero.ReadFile(m.fs, srcPath)

	if err != nil {
		return err
	}

	srcInfo, err := m.fs.Stat(srcPath)

	if err != nil {
		return err
	}

	err = afero.WriteFile(m.fs, targetPath, srcContents, srcInfo.Mode())

	if err != nil {
		return err
	}

	m.tui.Success(fmt.Sprintf("Successfully copied example file: %s->%s", srcPath, targetPath))

	return nil
}

func NewManager(fs afero.Fs, tui tui) *Manager {
	return &Manager{
		fs:  fs,
		tui: tui,
	}
}
