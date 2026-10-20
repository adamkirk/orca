package stack

import (
	"context"
	"fmt"

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

// startupOrder orders the projects to start: the one asked for, or the
// workspace's enabled projects and their requirements, requirements first.
func (m *Manager) startupOrder(execCtx workspace.ExecutionContext) ([]string, error) {
	if execCtx.Project != nil {
		if execCtx.Project.Disabled {
			m.tui.Info(fmt.Sprintf("Note: %s is disabled, but starting it as you asked for it.", execCtx.Project.Name))
		}

		return []string{
			execCtx.Project.Name,
		}, nil
	}

	projects, pulledIn := execCtx.Workspace.RunSet()

	for _, p := range pulledIn {
		m.tui.Info(fmt.Sprintf("Warning: %s requires %s, which is disabled, so it's being started anyway.", p.RequiredBy, p.Name))
	}

	g, err := dag.NewGraph(projects)

	if err != nil {
		return nil, err
	}

	return g.TopologicalKeysFromRoots()
}

func (m *Manager) startServices(execCtx workspace.ExecutionContext) error {
	ordered, err := m.startupOrder(execCtx)

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
