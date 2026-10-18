package stack

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

type ShowComposeConfigQuery struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (ShowComposeConfigQuery) MessageName() string {
	return "debug.show-compose-config"
}

func (m *Manager) ShowComposeConfig(ctx context.Context, query ShowComposeConfigQuery) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	if execCtx.Project == nil {
		return workspace.ErrInvalidExecutionContext{
			Msg: "this command must be run in a singular project context",
		}
	}

	return m.compose.ShowConfig(execCtx.Workspace, execCtx.Project)
}
