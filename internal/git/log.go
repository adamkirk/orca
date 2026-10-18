package git

import (
	"context"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
)

type LoglQuery struct {
	bus.Returns[bus.Void]

	// The amount of commits to show
	Amount int
	// The directory to run the command in, defaults to the current directory
	Dir string
}

func (LoglQuery) MessageName() string {
	return "g.logl"
}

func (g *Git) Logl(_ context.Context, query LoglQuery) error {
	if err := g.mustBeInAGitRepository(query.Dir); err != nil {
		return err
	}

	opt, stdout := process.WithStdout()
	err := g.exec.Exec("git", []string{
		"log",
		"--oneline",
	}, withDir(query.Dir, opt)...)

	if err != nil {
		return usererr.Wrap(err, "Something went wrong, this is most likely a bug!")
	}

	linesOutput := stdout.String()

	lines := strings.Split(linesOutput, "\n")

	lineCount := min(query.Amount, len(lines))

	output := strings.Join(lines[0:lineCount], "\n")

	g.tui.Info(output)

	return nil
}
