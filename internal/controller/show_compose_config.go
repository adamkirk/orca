package controller

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
)

type ShowComposeConfigQuery struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (ShowComposeConfigQuery) MessageName() string {
	return "debug.show-compose-config"
}

func (c *Controller) ShowComposeConfig(_ context.Context, query ShowComposeConfigQuery) error {
	ctx, err := c.contextResolver.Resolve(query.Workspace, query.Project)

	if err != nil {
		return err
	}

	if ctx.Project == nil {
		return common.ErrInvalidExecutionContext{
			Msg: "this command must be run in a singular project context",
		}
	}

	return c.compose.ShowConfig(ctx.Workspace, ctx.Project)
}
