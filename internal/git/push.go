package git

import (
	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/hostsys"
)

type PushBranchDTO struct {
	Force bool
	// The directory to run the command in, defaults to the current directory
	Dir string
}

func (g *Git) PushBranch(dto PushBranchDTO) error {
	if err := g.mustBeInAGitRepository(dto.Dir); err != nil {
		return err
	}

	branch, err := g.GetCurrentBranch(dto.Dir)

	if err != nil {
		return g.tui.RecordIfError("Failed to determine current branch, and no branch was supplied!", err)
	}

	args := []string{
		"push",
	}

	if dto.Force {
		args = append(args, "-f")
	}

	args = append(args, "origin", branch)

	return g.exec.Exec("git", args, withDir(dto.Dir, hostsys.WithHostIO())...)
}

type PushDTO struct {
	AllProjects bool
	Workspace   string
	Project     string
	Force       bool
}

// pushInContext pushes the current branch for the resolved project (or working
// directory), or for every project in the workspace when allProjects is set
func (g *Git) pushInContext(ctx common.ExecutionContext, allProjects bool, force bool) error {
	if !allProjects {
		if ctx.Project == nil {
			return g.PushBranch(PushBranchDTO{Dir: ctx.WorkingDirectory, Force: force})
		}

		return g.PushBranch(PushBranchDTO{Dir: ctx.Project.ProjectDir, Force: force})
	}

	for _, p := range ctx.Workspace.Projects {
		if err := g.PushBranch(PushBranchDTO{Dir: p.ProjectDir, Force: force}); err != nil {
			return g.tui.RecordIfError("Push failed, you should check the branches for all projects as this failed mid-flow", err)
		}
	}

	return nil
}

func (g *Git) Push(dto PushDTO) error {
	ctx, err := g.resolveContext(dto.Workspace, dto.Project, dto.AllProjects)

	if err != nil {
		return err
	}

	return g.pushInContext(ctx, dto.AllProjects, dto.Force)
}
