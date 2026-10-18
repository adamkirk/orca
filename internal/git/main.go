package git

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/hostsys"
)

const ConfirmYes = "Yes"
const ConfirmNo = "No"

type tui interface {
	Info(msg ...string)
	Error(msg ...string)
	Success(msg ...string)
	RecordIfError(msg string, err error) error
	NewLine()
	PresentChoices(opts []string, title string) (string, error)
	Table(header []string, rows [][]string)
}

type executor interface {
	Exec(cmdName string, args []string, opts ...hostsys.ExecOpt) error
}

// withDir prepends a ChdirOpt to the given opts when dir is not empty
func withDir(dir string, opts ...hostsys.ExecOpt) []hostsys.ExecOpt {
	if dir == "" {
		return opts
	}

	return append([]hostsys.ExecOpt{hostsys.ChdirOpt(dir)}, opts...)
}

type Git struct {
	exec executor
	tui  tui
}

func (g *Git) mustBeInAGitRepository(dir string) error {
	if g.isInGitRepository(dir) {
		return nil
	}

	return g.tui.RecordIfError(
		"You don't appear to be in a git repository!",
		common.ErrInvalidExecutionContext{
			Msg: "this command must be executed from within a git repository",
		},
	)

}

func (g *Git) isInGitRepository(dir string) bool {
	err := g.exec.Exec("git", []string{"rev-parse", "--is-inside-work-tree"}, withDir(dir)...)

	return err == nil
}

func (g *Git) GetRepositoryRootFromPath(path string) (string, error) {
	stdoutOpt, stdout := hostsys.WithStdout()
	stderrOpt, stderr := hostsys.WithStderr()

	dir := filepath.Dir(path)

	err := g.exec.Exec("git", []string{
		"rev-parse",
		"--show-toplevel",
	}, hostsys.ChdirOpt(dir), stdoutOpt, stderrOpt)

	if err != nil {
		errOutput := stderr.String()
		slog.Error("failed to execute 'git rev-parse --show-top-level'", "in", dir, "err", err, "stderr", errOutput)

		return "", common.ErrCommandExecutionFailed{
			Msg: errOutput,
		}
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (g *Git) DirectoryHasOrigin(dir string, origin string) (bool, error) {
	stdoutOpt, stdout := hostsys.WithStdout()

	err := g.exec.Exec("git", []string{
		"remote",
		"get-url",
		"origin",
	}, hostsys.ChdirOpt(dir), stdoutOpt)

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
