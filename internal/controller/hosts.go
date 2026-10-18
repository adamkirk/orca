package controller

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
)

type HostsCommand struct {
	bus.Returns[bus.Void]

	Workspace string
}

func (HostsCommand) MessageName() string {
	return "hosts"
}

func (c *Controller) Hosts(ctx context.Context, cmd HostsCommand) error {
	execCtx, err := common.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	ws, err := c.workspaceRepo.Load(execCtx.Workspace.Name)

	if err != nil {
		return err
	}

	if err := c.etcHostsManager.SyncForWorkspace(ws); err != nil {
		return c.tui.RecordIfError("Failed to save /etc/hosts file!", err)
	}

	c.tui.Success("Hosts are now added to the /etc/hosts file!")

	return nil
}
