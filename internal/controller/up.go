package controller

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/pkg/dag"
)

type UpCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (UpCommand) MessageName() string {
	return "up"
}

func determineStartupOrder(ctx common.ExecutionContext) ([]string, error) {
	if ctx.Project != nil {
		return []string{
			ctx.Project.Name,
		}, nil
	}

	g, err := dag.NewGraph(ctx.Workspace.Projects)

	if err != nil {
		return nil, err
	}

	return g.TopologicalKeysFromRoots()
}

func (c *Controller) startServices(ctx common.ExecutionContext) error {
	ordered, err := determineStartupOrder(ctx)

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

		if err := c.compose.Up(ctx.Workspace, projectConfig); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) Up(_ context.Context, cmd UpCommand) error {
	ctx, err := c.contextResolver.Resolve(cmd.Workspace, cmd.Project)

	if err != nil {
		return err
	}

	return c.startServices(ctx)
}
