package controller

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
)

type LogsQuery struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
	Service   string
}

func (LogsQuery) MessageName() string {
	return "logs"
}

func (c *Controller) Logs(_ context.Context, query LogsQuery) error {
	ctx, err := c.contextResolver.Resolve(query.Workspace, query.Project)

	if err != nil {
		return err
	}

	return c.compose.Logs(ctx.Workspace, ctx.Project, query.Service)
}
