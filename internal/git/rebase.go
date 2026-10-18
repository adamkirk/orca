package git

import (
	"context"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/process"
)

type RebaseInteractivelyCommand struct {
	bus.Returns[bus.Void]

	// The amount of commits to include in the rebase
	Amount int
	// The directory to run the command in, defaults to the current directory
	Dir string
}

func (RebaseInteractivelyCommand) MessageName() string {
	return "g.rbi"
}

func (g *Git) RebaseInteractively(_ context.Context, cmd RebaseInteractivelyCommand) error {
	if err := g.mustBeInAGitRepository(cmd.Dir); err != nil {
		return err
	}

	return g.exec.Exec("git", []string{
		"rebase",
		"-i",
		fmt.Sprintf("HEAD~%d", cmd.Amount),
	}, withDir(cmd.Dir, process.WithHostIO())...)
}
