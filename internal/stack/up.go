package stack

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/dag"
	"github.com/adamkirk/orca/internal/workspace"
)

type UpCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (UpCommand) MessageName() string {
	return "up"
}

func determineStartupOrder(execCtx workspace.ExecutionContext) ([]string, error) {
	if execCtx.Project != nil {
		return []string{
			execCtx.Project.Name,
		}, nil
	}

	g, err := dag.NewGraph(execCtx.Workspace.Projects)

	if err != nil {
		return nil, err
	}

	return g.TopologicalKeysFromRoots()
}

func (m *Manager) startServices(execCtx workspace.ExecutionContext) error {
	ordered, err := determineStartupOrder(execCtx)

	if err != nil {
		return err
	}

	for _, p := range ordered {
		// This shouldn't ever really return an error at this point, if it does
		// something went wrong while building the runtime context
		projectConfig, err := execCtx.Workspace.GetProject(p)

		if err != nil {
			return err
		}

		if err := m.compose.Up(execCtx.Workspace, projectConfig); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Up(ctx context.Context, cmd UpCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return m.startServices(execCtx)
}
