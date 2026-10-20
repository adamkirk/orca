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

	// Run every extension the hooks run without asking, and without recording
	// them as trusted
	Yes bool
}

func (UpCommand) MessageName() string {
	return "up"
}

// startupOrder orders the projects to start: the one asked for, or the
// workspace's enabled projects and their requirements, requirements first.
// Libraries are never started.
func (m *Manager) startupOrder(execCtx workspace.ExecutionContext) ([]string, error) {
	if execCtx.Project != nil {
		if execCtx.Project.Disabled {
			m.tui.Info(fmt.Sprintf("Note: %s is disabled, but starting it as you asked for it.", execCtx.Project.Name))
		}

		m.warnAboutUnclonedLibraries(execCtx.Workspace, []workspace.Project{*execCtx.Project})

		return []string{
			execCtx.Project.Name,
		}, nil
	}

	projects, pulledIn := execCtx.Workspace.RunSet()

	for _, p := range pulledIn {
		m.tui.Info(fmt.Sprintf("Warning: %s requires %s, which is disabled, so it's being started anyway.", p.RequiredBy, p.Name))
	}

	m.warnAboutUnclonedLibraries(execCtx.Workspace, projects)

	// Requirements on libraries are left out, as they aren't started
	g, err := dag.NewGraph(workspace.TrimRequires(projects))

	if err != nil {
		return nil, err
	}

	return g.TopologicalKeysFromRoots()
}

// warnAboutUnclonedLibraries warns about each library the projects require
// that isn't cloned, e.g. as it's been added since the workspace was cloned.
// The projects are started anyway, as they may work without it.
func (m *Manager) warnAboutUnclonedLibraries(ws *workspace.Workspace, projects []workspace.Project) {
	for _, p := range projects {
		for _, name := range p.Requires {
			required, err := ws.GetProject(name)

			if err != nil || !required.IsLibrary() || required.IsRegistered {
				continue
			}

			m.tui.Info(fmt.Sprintf("Warning: %s requires %s, which isn't cloned; clone it with 'orca ws clone -p %s'.", p.Name, name, name))
		}
	}
}

// startServices starts the projects, each between its up.pre and up.post
// hooks, and for the whole workspace, between the workspace's too.
func (m *Manager) startServices(execCtx workspace.ExecutionContext, yes bool) error {
	ordered, err := m.startupOrder(execCtx)

	if err != nil {
		return err
	}

	return m.withHooks(execCtx, ordered, workspace.HookUpPre, workspace.HookUpPost, yes, func(p *workspace.Project) error {
		return m.compose.Up(execCtx.Workspace, p)
	})
}

func (m *Manager) Up(ctx context.Context, cmd UpCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return m.startServices(execCtx, cmd.Yes)
}
