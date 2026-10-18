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

func (c *Controller) ShowComposeConfig(ctx context.Context, query ShowComposeConfigQuery) error {
	execCtx, err := common.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	if execCtx.Project == nil {
		return common.ErrInvalidExecutionContext{
			Msg: "this command must be run in a singular project context",
		}
	}

	return c.compose.ShowConfig(execCtx.Workspace, execCtx.Project)
}
