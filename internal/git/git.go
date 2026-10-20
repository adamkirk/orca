package git

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
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

type Manager struct {
	exec executor
	tui  tui
}

func (m *Manager) mustBeInAGitRepository(dir string) error {
	if m.isInGitRepository(dir) {
		return nil
	}

	return usererr.Wrap(
		fmt.Errorf("%w: %s", workspace.ErrInvalidExecutionContext, "this command must be executed from within a git repository"),
		"You don't appear to be in a git repository!",
	)

}

func (m *Manager) isInGitRepository(dir string) bool {
	err := m.exec.Exec("git", []string{"rev-parse", "--is-inside-work-tree"}, withDir(dir)...)

	return err == nil
}

func (m *Manager) GetRepositoryRootFromPath(path string) (string, error) {
	stdoutOpt, stdout := process.WithStdout()
	stderrOpt, stderr := process.WithStderr()

	dir := filepath.Dir(path)

	err := m.exec.Exec("git", []string{
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

func (m *Manager) DirectoryHasOrigin(dir string, origin string) (bool, error) {
	stdoutOpt, stdout := process.WithStdout()

	err := m.exec.Exec("git", []string{
		"remote",
		"get-url",
		"origin",
	}, process.ChdirOpt(dir), stdoutOpt)

	if err != nil {
		return false, err
	}

	return strings.TrimSpace(stdout.String()) == origin, nil
}

// IsIgnored reports whether git ignores the file at path, in the repository
// that contains it.
func (m *Manager) IsIgnored(path string) (bool, error) {
	err := m.exec.Exec("git", []string{"check-ignore", "-q", filepath.Base(path)}, process.ChdirOpt(filepath.Dir(path)))

	if err == nil {
		return true, nil
	}

	// check-ignore exits with 1 when the file isn't ignored, and 128 for errors
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}

	return false, fmt.Errorf("checking whether git ignores %s: %w", path, err)
}

func NewManager(exec executor, tui tui) *Manager {
	return &Manager{
		exec: exec,
		tui:  tui,
	}
}
