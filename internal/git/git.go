package git

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

const ConfirmYes = "Yes"
const ConfirmNo = "No"

type tui interface {
	Info(msg ...string)
	Success(msg ...string)
	NewLine()
	PresentChoices(opts []string, title string) (string, error)
	Table(header []string, rows [][]string)
}

type executor interface {
	Exec(cmdName string, args []string, opts ...process.ExecOpt) error
}

// withDir prepends a ChdirOpt to the given opts when dir is not empty
func withDir(dir string, opts ...process.ExecOpt) []process.ExecOpt {
	if dir == "" {
		return opts
	}

	return append([]process.ExecOpt{process.ChdirOpt(dir)}, opts...)
}

type Git struct {
	exec executor
	tui  tui
}

func (g *Git) mustBeInAGitRepository(dir string) error {
	if g.isInGitRepository(dir) {
		return nil
	}

	return usererr.Wrap(
		fmt.Errorf("%w: %s", workspace.ErrInvalidExecutionContext, "this command must be executed from within a git repository"),
		"You don't appear to be in a git repository!",
	)

}

func (g *Git) isInGitRepository(dir string) bool {
	err := g.exec.Exec("git", []string{"rev-parse", "--is-inside-work-tree"}, withDir(dir)...)

	return err == nil
}

func (g *Git) GetRepositoryRootFromPath(path string) (string, error) {
	stdoutOpt, stdout := process.WithStdout()
	stderrOpt, stderr := process.WithStderr()

	dir := filepath.Dir(path)

	err := g.exec.Exec("git", []string{
		"rev-parse",
		"--show-toplevel",
	}, process.ChdirOpt(dir), stdoutOpt, stderrOpt)

	if err != nil {
		errOutput := stderr.String()
		slog.Error("failed to execute 'git rev-parse --show-top-level'", "in", dir, "err", err, "stderr", errOutput)

		return "", fmt.Errorf("%w: %s", ErrCommandExecutionFailed, errOutput)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (g *Git) DirectoryHasOrigin(dir string, origin string) (bool, error) {
	stdoutOpt, stdout := process.WithStdout()

	err := g.exec.Exec("git", []string{
		"remote",
		"get-url",
		"origin",
	}, process.ChdirOpt(dir), stdoutOpt)

	if err != nil {
		return false, err
	}

	return strings.TrimSpace(stdout.String()) == origin, nil
}

func NewGit(exec executor, tui tui) *Git {
	return &Git{
		exec: exec,
		tui:  tui,
	}
}
