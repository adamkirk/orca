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

func (c *Controller) ShowComposeCommand(ctx context.Context, query ShowComposeCommandQuery) error {
	execCtx, err := common.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	if execCtx.Project == nil {
		return common.ErrInvalidExecutionContext{
			Msg: "this command must be run in a singular project context",
		}
	}

	return c.compose.ShowCommand(execCtx.Workspace, execCtx.Project)
}
