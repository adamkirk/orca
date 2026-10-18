package controller

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
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

func (c *Controller) Logs(ctx context.Context, query LogsQuery) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return c.compose.Logs(execCtx.Workspace, execCtx.Project, query.Service)
}
