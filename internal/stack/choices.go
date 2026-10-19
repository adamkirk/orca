package stack

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

// ServiceChoicesQuery lists the services in the project's compose file.
type ServiceChoicesQuery struct {
	bus.Returns[[]workspace.Choice]

	Workspace string
	Project   string
}

func (ServiceChoicesQuery) MessageName() string {
	return "choices.services"
}

func (m *Manager) ServiceChoices(ctx context.Context, _ ServiceChoicesQuery) ([]workspace.Choice, error) {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return nil, err
	}

	// Outside a project, there's no one compose file to read
	if execCtx.Project == nil {
		return nil, ErrNoProject
	}

	names, err := m.compose.ServiceNames(execCtx.Project)

	if err != nil {
		return nil, err
	}

	choices := make([]workspace.Choice, len(names))

	for i, name := range names {
		choices[i] = workspace.Choice{Value: name}
	}

	return choices, nil
}
