package git

import (
	"context"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/hostsys"
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

	opt, stdout := hostsys.WithStdout()
	err := g.exec.Exec("git", []string{
		"log",
		"--oneline",
	}, withDir(query.Dir, opt)...)

	if err != nil {
		return g.tui.RecordIfError("Something went wrong, this is most likely a bug!", err)
	}

	linesOutput := stdout.String()

	lines := strings.Split(linesOutput, "\n")

	lineCount := min(query.Amount, len(lines))

	output := strings.Join(lines[0:lineCount], "\n")

	g.tui.Info(output)

	return nil
}
