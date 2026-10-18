package controller

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

func (c *Controller) ExecOrRun(ctx context.Context, cmd ExecCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	svcExists, err := c.compose.DoesSvcExist(execCtx.Workspace, execCtx.Project, cmd.Service)

	if err != nil {
		return err
	}

	if !svcExists {
		c.tui.Error(fmt.Sprintf("No such service: %s", cmd.Service))
		return ErrUnknownService{
			Name: cmd.Service,
		}
	}

	isRunning, err := c.compose.IsSvcRunning(execCtx.Workspace, execCtx.Project, cmd.Service)

	if err != nil {
		return err
	}

	if !isRunning {
		return c.compose.Run(execCtx.Workspace, execCtx.Project, cmd.Service, cmd.Args)
	}

	return c.compose.Exec(execCtx.Workspace, execCtx.Project, cmd.Service, cmd.Args)
}
