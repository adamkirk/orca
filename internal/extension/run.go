package extension

import (
	"context"
	"errors"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// RunCommand runs an extension by name: the project's, or the workspace's
// when the project doesn't define it, or outside of any project.
type RunCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
	Name      string
	Args      []string
}

func (RunCommand) MessageName() string {
	return "ext"
}

func (m *Manager) Run(ctx context.Context, cmd RunCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	ext, err := m.find(execCtx, cmd.Name)

	if err != nil {
		return err
	}

	t := target{ws: execCtx.Workspace, project: execCtx.Project}

	return usererr.Wrap(m.execute(t, ext, cmd.Args), fmt.Sprintf("Extension '%s' failed!", cmd.Name))
}

// find looks for the extension in the project first, then the workspace, so a
// project can override one of the workspace's.
func (m *Manager) find(execCtx workspace.ExecutionContext, name string) (workspace.Extension, error) {
	wsExt, wsErr := execCtx.Workspace.FindExtension(name)

	if execCtx.Project != nil {
		ext, err := execCtx.Project.FindExtension(name)

		if err == nil {
			if wsErr == nil {
				m.tui.Info(fmt.Sprintf("Note: running %s's own '%s', rather than the workspace's.", execCtx.Project.Name, name))
			}

			return ext, nil
		}

		if !errors.Is(err, workspace.ErrUnknownExtension) {
			return workspace.Extension{}, err
		}
	}

	if wsErr != nil {
		where := "the workspace"

		if execCtx.Project != nil {
			where = fmt.Sprintf("%s or the workspace", execCtx.Project.Name)
		}

		return workspace.Extension{}, usererr.Wrap(wsErr, fmt.Sprintf("There's no extension '%s' in %s!", name, where))
	}

	return wsExt, nil
}
