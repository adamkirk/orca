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

	run, err := m.find(execCtx, cmd.Name)

	if err != nil {
		return err
	}

	// Asking for it by name is trusting it, so it isn't asked about when a hook
	// runs it
	if err := m.trust.Trust(map[TrustKey]Trusted{run.trustKey(): trusted(run.ext)}); err != nil {
		return usererr.Wrap(err, "Failed to record the extension as trusted!")
	}

	return usererr.Wrap(m.execute(run.target, run.ext, cmd.Args, ""), fmt.Sprintf("Extension '%s' failed!", cmd.Name))
}

// find looks for the extension in the project first, then the workspace, so a
// project can override one of the workspace's. Either way, it runs in the
// project, if there is one.
func (m *Manager) find(execCtx workspace.ExecutionContext, name string) (planned, error) {
	t := target{ws: execCtx.Workspace, project: execCtx.Project}
	wsExt, wsErr := execCtx.Workspace.FindExtension(name)

	if execCtx.Project != nil {
		ext, err := execCtx.Project.FindExtension(name)

		if err == nil {
			if wsErr == nil {
				m.tui.Info(fmt.Sprintf("Note: running %s's own '%s', rather than the workspace's.", execCtx.Project.Name, name))
			}

			return planned{owner: execCtx.Project, target: t, ext: ext}, nil
		}

		if !errors.Is(err, workspace.ErrUnknownExtension) {
			return planned{}, err
		}
	}

	if wsErr != nil {
		where := "the workspace"

		if execCtx.Project != nil {
			where = fmt.Sprintf("%s or the workspace", execCtx.Project.Name)
		}

		return planned{}, usererr.Wrap(wsErr, fmt.Sprintf("There's no extension '%s' in %s!", name, where))
	}

	return planned{target: t, ext: wsExt}, nil
}
