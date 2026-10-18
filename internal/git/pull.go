package git

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

type PullCommand struct {
	bus.Returns[bus.Void]

	AllProjects bool
	Workspace   string
	Project     string
	Rebase      bool
}

func (PullCommand) MessageName() string {
	return "g.pull"
}

// pullInContext pulls the current branch for the resolved project (or working
// directory), or for every project in the workspace when allProjects is set
func (g *Git) pullInContext(execCtx workspace.ExecutionContext, allProjects bool, rebase bool) error {
	if !allProjects {
		return g.PullBranch(PullBranchOpts{Dir: execCtx.Dir(), Rebase: rebase})
	}

	for _, p := range execCtx.Workspace.Projects {
		if err := g.PullBranch(PullBranchOpts{Dir: p.ProjectDir, Rebase: rebase}); err != nil {
			return err
		}
	}

	return nil
}

func (g *Git) Pull(ctx context.Context, cmd PullCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return g.pullInContext(execCtx, cmd.AllProjects, cmd.Rebase)
}
