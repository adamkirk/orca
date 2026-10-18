package git

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
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
func (g *Git) pullInContext(ctx common.ExecutionContext, allProjects bool, rebase bool) error {
	if !allProjects {
		if ctx.Project == nil {
			return g.PullBranch(PullBranchDTO{Dir: ctx.WorkingDirectory, Rebase: rebase})
		}

		return g.PullBranch(PullBranchDTO{Dir: ctx.Project.ProjectDir, Rebase: rebase})
	}

	for _, p := range ctx.Workspace.Projects {
		if err := g.PullBranch(PullBranchDTO{Dir: p.ProjectDir, Rebase: rebase}); err != nil {
			return err
		}
	}

	return nil
}

func (g *Git) Pull(_ context.Context, cmd PullCommand) error {
	ctx, err := g.resolveContext(cmd.Workspace, cmd.Project, cmd.AllProjects)

	if err != nil {
		return err
	}

	return g.pullInContext(ctx, cmd.AllProjects, cmd.Rebase)
}
