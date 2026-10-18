package git

import (
	"context"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/process"
	orcatui "github.com/adamkirk/orca/internal/tui"
)

type UndoLastXCommitsCommand struct {
	bus.Returns[bus.Void]

	// The amount of commits to remoe
	Amount           int
	SkipConfirmation bool
	// The directory to run the command in, defaults to the current directory
	Dir string
}

func (UndoLastXCommitsCommand) MessageName() string {
	return "g.undo"
}

func (g *Git) UndoLastXCommits(_ context.Context, cmd UndoLastXCommitsCommand) error {
	if err := g.mustBeInAGitRepository(cmd.Dir); err != nil {
		return err
	}

	if !cmd.SkipConfirmation {
		result, err := g.tui.PresentChoices([]string{
			ConfirmNo,
			ConfirmYes,
		}, "You might lose some precious changes, are you sure?")

		if err != nil {
			if _, ok := err.(orcatui.ErrUserAbortedExecution); ok {
				return err
			}

			return g.tui.RecordIfError("Something went wrong, this is most likely a bug!", err)
		}

		if result == ConfirmNo {
			g.tui.Info("Aborting")
			return nil
		}
	}

	return g.exec.Exec("git", []string{
		"reset",
		"--hard",
		fmt.Sprintf("HEAD~%d", cmd.Amount),
	}, withDir(cmd.Dir, process.WithHostIO())...)
}
