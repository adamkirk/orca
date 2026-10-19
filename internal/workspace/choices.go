package workspace

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
)

// Choice is a value the user can choose for an argument or flag, offered by
// shell completion.
type Choice struct {
	Value string
	// Shown alongside the value where the shell supports it, may be empty
	Description string
}

// WorkspaceChoicesQuery lists the registered workspaces.
type WorkspaceChoicesQuery struct {
	bus.Returns[[]Choice]
}

func (WorkspaceChoicesQuery) MessageName() string {
	return "choices.workspaces"
}

func (m *Manager) WorkspaceChoices(_ context.Context, _ WorkspaceChoicesQuery) ([]Choice, error) {
	metas := m.configManager.GetAllWorkspaceMeta()
	choices := make([]Choice, len(metas))

	for i, meta := range metas {
		choices[i] = Choice{Value: meta.Name}
	}

	return choices, nil
}

// ProjectChoicesQuery lists the projects in the workspace, which is worked out
// as it is for any command, from the working directory or Workspace.
type ProjectChoicesQuery struct {
	bus.Returns[[]Choice]

	Workspace string
}

func (ProjectChoicesQuery) MessageName() string {
	return "choices.projects"
}

func (m *Manager) ProjectChoices(ctx context.Context, _ ProjectChoicesQuery) ([]Choice, error) {
	execCtx, err := ExecutionContextFrom(ctx)

	if err != nil {
		return nil, err
	}

	choices := make([]Choice, len(execCtx.Workspace.Projects))

	for i, p := range execCtx.Workspace.Projects {
		choices[i] = Choice{Value: p.Name}
	}

	return choices, nil
}
