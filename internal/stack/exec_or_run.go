package stack

import (
	"context"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

type ExecCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
	Service   string
	Args      []string
}

func (ExecCommand) MessageName() string {
	return "exec"
}

func (m *Manager) ExecOrRun(ctx context.Context, cmd ExecCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	svcExists, err := m.compose.DoesSvcExist(execCtx.Workspace, execCtx.Project, cmd.Service)

	if err != nil {
		return err
	}

	if !svcExists {
		m.tui.Error(fmt.Sprintf("No such service: %s", cmd.Service))
		return ErrUnknownService{
			Name: cmd.Service,
		}
	}

	isRunning, err := m.compose.IsSvcRunning(execCtx.Workspace, execCtx.Project, cmd.Service)

	if err != nil {
		return err
	}

	if !isRunning {
		return m.compose.Run(execCtx.Workspace, execCtx.Project, cmd.Service, cmd.Args)
	}

	return m.compose.Exec(execCtx.Workspace, execCtx.Project, cmd.Service, cmd.Args)
}
