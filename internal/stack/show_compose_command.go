package stack

import (
	"context"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

type ShowComposeCommandQuery struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (ShowComposeCommandQuery) MessageName() string {
	return "debug.show-compose-command"
}

func (m *Manager) ShowComposeCommand(ctx context.Context, query ShowComposeCommandQuery) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	if execCtx.Project == nil {
		return fmt.Errorf("%w: %s", workspace.ErrInvalidExecutionContext, "this command must be run in a singular project context")
	}

	return m.compose.ShowCommand(execCtx.Workspace, execCtx.Project)
}
