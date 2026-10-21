package extension

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

// ChoicesQuery lists the project's extensions, with their descriptions.
type ChoicesQuery struct {
	bus.Returns[[]workspace.Choice]

	Workspace string
	Project   string
}

func (ChoicesQuery) MessageName() string {
	return "choices.extensions"
}

func (m *Manager) Choices(ctx context.Context, _ ChoicesQuery) ([]workspace.Choice, error) {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return nil, err
	}

	choices := []workspace.Choice{}

	// Outside a project, there's nothing to run
	if execCtx.Project == nil {
		return choices, nil
	}

	for _, ext := range execCtx.Project.Config.Extensions {
		choices = append(choices, workspace.Choice{Value: ext.Name, Description: ext.Description})
	}

	return choices, nil
}
