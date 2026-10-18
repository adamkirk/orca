package git

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/hostsys"
)

type PushBranchOpts struct {
	Force bool
	// The directory to run the command in, defaults to the current directory
	Dir string
}

func (g *Git) PushBranch(cmd PushBranchOpts) error {
	if err := g.mustBeInAGitRepository(cmd.Dir); err != nil {
		return err
	}

	branch, err := g.GetCurrentBranch(cmd.Dir)

	if err != nil {
		return g.tui.RecordIfError("Failed to determine current branch, and no branch was supplied!", err)
	}

	args := []string{
		"push",
	}

	if cmd.Force {
		args = append(args, "-f")
	}

	args = append(args, "origin", branch)

	return g.exec.Exec("git", args, withDir(cmd.Dir, hostsys.WithHostIO())...)
}

type PushCommand struct {
	bus.Returns[bus.Void]

	AllProjects bool
	Workspace   string
	Project     string
	Force       bool
}

func (PushCommand) MessageName() string {
	return "g.push"
}

// pushInContext pushes the current branch for the resolved project (or working
// directory), or for every project in the workspace when allProjects is set
func (g *Git) pushInContext(execCtx common.ExecutionContext, allProjects bool, force bool) error {
	if !allProjects {
		return g.PushBranch(PushBranchOpts{Dir: execCtx.Dir(), Force: force})
	}

	for _, p := range execCtx.Workspace.Projects {
		if err := g.PushBranch(PushBranchOpts{Dir: p.ProjectDir, Force: force}); err != nil {
			return g.tui.RecordIfError("Push failed, you should check the branches for all projects as this failed mid-flow", err)
		}
	}

	return nil
}

func (g *Git) Push(ctx context.Context, cmd PushCommand) error {
	execCtx, err := common.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return g.pushInContext(execCtx, cmd.AllProjects, cmd.Force)
}
