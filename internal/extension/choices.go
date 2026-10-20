package extension

import (
	"context"
	"slices"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

// ChoicesQuery lists the extensions that can be run in the context, with their
// descriptions: the project's, then the workspace's that it doesn't override.
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

	if execCtx.Project != nil {
		for _, ext := range execCtx.Project.Config.Extensions {
			choices = append(choices, workspace.Choice{Value: ext.Name, Description: ext.Description})
		}
	}

	for _, ext := range execCtx.Workspace.Extensions {
		overridden := slices.ContainsFunc(choices, func(c workspace.Choice) bool {
			return c.Value == ext.Name
		})

		if !overridden {
			choices = append(choices, workspace.Choice{Value: ext.Name, Description: ext.Description})
		}
	}

	return choices, nil
}
