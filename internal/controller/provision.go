package controller

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

type ProvisionCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (ProvisionCommand) MessageName() string {
	return "provision"
}

func (c *Controller) Provision(ctx context.Context, cmd ProvisionCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return c.provisionerRunner.RunAll(execCtx.Workspace, execCtx.Project)
}
