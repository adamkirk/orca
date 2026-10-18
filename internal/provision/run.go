package provision

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

type RunCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (RunCommand) MessageName() string {
	return "provision"
}

func (p *Runner) Provision(ctx context.Context, cmd RunCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return p.RunAll(execCtx.Workspace, execCtx.Project)
}
