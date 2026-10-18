package controller

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
)

type HostsCommand struct {
	bus.Returns[bus.Void]

	Workspace string
}

func (HostsCommand) MessageName() string {
	return "hosts"
}

func (c *Controller) Hosts(_ context.Context, cmd HostsCommand) error {
	ctx, err := c.contextResolver.Resolve(cmd.Workspace, "")

	if err != nil {
		return err
	}

	ws, err := c.workspaceRepo.Load(ctx.Workspace.Name)

	if err != nil {
		return err
	}

	if err := c.etcHostsManager.SyncForWorkspace(ws); err != nil {
		return c.tui.RecordIfError("Failed to save /etc/hosts file!", err)
	}

	c.tui.Success("Hosts are now added to the /etc/hosts file!")

	return nil
}
