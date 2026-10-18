package controller

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
)

type ProvisionCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (ProvisionCommand) MessageName() string {
	return "provision"
}

func (c *Controller) Provision(_ context.Context, cmd ProvisionCommand) error {
	ctx, err := c.contextResolver.Resolve(cmd.Workspace, cmd.Project)

	if err != nil {
		return err
	}

	return c.provisionerRunner.RunAll(ctx.Workspace, ctx.Project)
}
