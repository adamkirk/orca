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

func determineStartupOrder(execCtx common.ExecutionContext) ([]string, error) {
	if execCtx.Project != nil {
		return []string{
			execCtx.Project.Name,
		}, nil
	}

	g, err := dag.NewGraph(execCtx.Workspace.Projects)

	if err != nil {
		return nil, err
	}

	return g.TopologicalKeysFromRoots()
}

func (c *Controller) startServices(execCtx common.ExecutionContext) error {
	ordered, err := determineStartupOrder(execCtx)

	if err != nil {
		return err
	}

	for _, p := range ordered {
		// This shouldn't ever really return an error at this point, if it does
		// something went wrong while building the runtime context
		projectConfig, err := execCtx.Workspace.GetProject(p)

		if err != nil {
			return err
		}

		if err := c.compose.Up(execCtx.Workspace, projectConfig); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) Up(ctx context.Context, cmd UpCommand) error {
	execCtx, err := common.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return c.startServices(execCtx)
}
