package controller

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
)

type ShowComposeCommandQuery struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (ShowComposeCommandQuery) MessageName() string {
	return "debug.show-compose-command"
}

func (c *Controller) ShowComposeCommand(_ context.Context, query ShowComposeCommandQuery) error {
	ctx, err := c.contextResolver.Resolve(query.Workspace, query.Project)

	if err != nil {
		return err
	}

	if ctx.Project == nil {
		return common.ErrInvalidExecutionContext{
			Msg: "this command must be run in a singular project context",
		}
	}

	return c.compose.ShowCommand(ctx.Workspace, ctx.Project)
}
