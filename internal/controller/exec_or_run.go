package controller

import (
	"context"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
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

func (c *Controller) ExecOrRun(_ context.Context, cmd ExecCommand) error {
	ctx, err := c.contextResolver.Resolve(cmd.Workspace, cmd.Project)

	if err != nil {
		return err
	}

	svcExists, err := c.compose.DoesSvcExist(ctx.Workspace, ctx.Project, cmd.Service)

	if err != nil {
		return err
	}

	if !svcExists {
		c.tui.Error(fmt.Sprintf("No such service: %s", cmd.Service))
		return common.ErrUnknownService{
			Name: cmd.Service,
		}
	}

	isRunning, err := c.compose.IsSvcRunning(ctx.Workspace, ctx.Project, cmd.Service)

	if err != nil {
		return err
	}

	if !isRunning {
		return c.compose.Run(ctx.Workspace, ctx.Project, cmd.Service, cmd.Args)
	}

	return c.compose.Exec(ctx.Workspace, ctx.Project, cmd.Service, cmd.Args)
}
