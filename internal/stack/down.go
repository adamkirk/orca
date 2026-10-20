package stack

import (
	"context"
	"fmt"
	"slices"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/dag"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

type DownCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (DownCommand) MessageName() string {
	return "down"
}

// shutdownOrder orders the projects to stop: the one asked for, or those a
// workspace-wide up starts, plus any disabled project that's still running.
// Projects are stopped before the projects they require.
func (m *Manager) shutdownOrder(execCtx workspace.ExecutionContext) ([]string, error) {
	if execCtx.Project != nil {
		return []string{
			execCtx.Project.Name,
		}, nil
	}

	ws := execCtx.Workspace
	projects, _ := ws.RunSet()

	for _, p := range ws.Projects {
		if p.IsLibrary() || slices.ContainsFunc(projects, func(r workspace.Project) bool { return r.Name == p.Name }) {
			continue
		}

		running, err := m.compose.IsProjectRunning(ws, &p)

		if err != nil {
			return nil, usererr.Wrap(err, fmt.Sprintf("Failed to check whether %s is running!", p.Name))
		}

		if running {
			projects = append(projects, p)
		}
	}

	g, err := dag.NewGraph(workspace.TrimRequires(projects))

	if err != nil {
		return nil, err
	}

	return g.TopologicalKeysFromLeaves()
}

func (m *Manager) stopServices(execCtx workspace.ExecutionContext) error {
	ordered, err := m.shutdownOrder(execCtx)

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

		if err := m.compose.Down(execCtx.Workspace, projectConfig); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Down(ctx context.Context, cmd DownCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return m.stopServices(execCtx)
}
