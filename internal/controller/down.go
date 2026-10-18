package controller

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/pkg/dag"
)

type DownCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (DownCommand) MessageName() string {
	return "down"
}

func determineShutdownOrder(ctx common.ExecutionContext) ([]string, error) {
	if ctx.Project != nil {
		return []string{
			ctx.Project.Name,
		}, nil
	}

	g, err := dag.NewGraph(ctx.Workspace.Projects)

	if err != nil {
		return nil, err
	}

	return g.TopologicalKeysFromLeaves()
}

func (c *Controller) stopServices(ctx common.ExecutionContext) error {
	ordered, err := determineShutdownOrder(ctx)

	if err != nil {
		return err
	}

	for _, p := range ordered {
		// This shouldn't ever really return an error at this point, if it does
		// something went wrong while building the runtime context
		projectConfig, err := ctx.Workspace.GetProject(p)

		if err != nil {
			return err
		}

		if err := c.compose.Down(ctx.Workspace, projectConfig); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) Down(_ context.Context, cmd DownCommand) error {
	ctx, err := c.contextResolver.Resolve(cmd.Workspace, cmd.Project)

	if err != nil {
		return err
	}

	return c.stopServices(ctx)
}
