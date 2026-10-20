package stack

import (
	"github.com/adamkirk/orca/internal/workspace"
)

// withHooks does op to each of the projects, in order, between the project's
// pre and post hooks. For the whole workspace, the workspace's pre hook runs
// first, and its post hook last. Every extension they run is asked about
// before anything is done, and it all stops at the first failure.
func (m *Manager) withHooks(execCtx workspace.ExecutionContext, names []string, pre workspace.Hook, post workspace.Hook, yes bool, op func(*workspace.Project) error) error {
	ws := execCtx.Workspace
	wholeWorkspace := execCtx.Project == nil
	projects := make([]*workspace.Project, len(names))
	runs := []workspace.HookRun{}

	if wholeWorkspace {
		runs = append(runs, workspace.HookRun{Hook: pre})
	}

	for i, name := range names {
		// This shouldn't ever really return an error at this point, if it does
		// something went wrong while building the runtime context
		p, err := ws.GetProject(name)

		if err != nil {
			return err
		}

		projects[i] = p
		runs = append(runs, workspace.HookRun{Project: p, Hook: pre}, workspace.HookRun{Project: p, Hook: post})
	}

	if wholeWorkspace {
		runs = append(runs, workspace.HookRun{Hook: post})
	}

	if err := m.hooks.ConfirmHooks(ws, runs, yes); err != nil {
		return err
	}

	if wholeWorkspace {
		if err := m.hooks.RunConfirmedHook(ws, workspace.HookRun{Hook: pre}); err != nil {
			return err
		}
	}

	for _, p := range projects {
		if err := m.hooks.RunConfirmedHook(ws, workspace.HookRun{Project: p, Hook: pre}); err != nil {
			return err
		}

		if err := op(p); err != nil {
			return err
		}

		if err := m.hooks.RunConfirmedHook(ws, workspace.HookRun{Project: p, Hook: post}); err != nil {
			return err
		}
	}

	if wholeWorkspace {
		return m.hooks.RunConfirmedHook(ws, workspace.HookRun{Hook: post})
	}

	return nil
}
