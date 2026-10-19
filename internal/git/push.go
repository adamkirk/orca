package git

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

type PushBranchOpts struct {
	Force bool
	// The directory to run the command in, defaults to the current directory
	Dir string
}

func (m *Manager) PushBranch(cmd PushBranchOpts) error {
	if err := m.mustBeInAGitRepository(cmd.Dir); err != nil {
		return err
	}

	branch, err := m.GetCurrentBranch(cmd.Dir)

	if err != nil {
		return usererr.Wrap(err, "Failed to determine current branch, and no branch was supplied!")
	}

	args := []string{
		"push",
	}

	if cmd.Force {
		args = append(args, "-f")
	}

	args = append(args, "origin", branch)

	return m.exec.Exec("git", args, withDir(cmd.Dir, process.WithHostIO())...)
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
func (m *Manager) pushInContext(execCtx workspace.ExecutionContext, allProjects bool, force bool) error {
	if !allProjects {
		return m.PushBranch(PushBranchOpts{Dir: execCtx.Dir(), Force: force})
	}

	for _, p := range execCtx.Workspace.Projects {
		if err := m.PushBranch(PushBranchOpts{Dir: p.ProjectDir, Force: force}); err != nil {
			return usererr.Wrap(err, "Push failed, you should check the branches for all projects as this failed mid-flow")
		}
	}

	return nil
}

func (m *Manager) Push(ctx context.Context, cmd PushCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	return m.pushInContext(execCtx, cmd.AllProjects, cmd.Force)
}
