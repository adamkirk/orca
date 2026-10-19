package git

import (
	"fmt"
	"strings"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/workspace"
)

// Clone clones the repository into target, checking out the given branch, or
// the remote's default branch when it's empty.
func (m *Manager) Clone(repoURL string, target string, branch string) error {
	if repoURL == "" || target == "" {
		return fmt.Errorf("%w to 'git.clone': 'repoURL' and 'target' cannot be empty", workspace.ErrInvalidInput)
	}

	args := []string{"clone"}

	if branch != "" {
		args = append(args, "--branch", branch)
	}

	return m.exec.Exec(
		"git",
		append(args, repoURL, target),
		process.WithHostIO(),
	)
}

// RemoteBranchExists checks whether the branch exists on the remote
// repository, without cloning it.
func (m *Manager) RemoteBranchExists(repoURL string, branch string) (bool, error) {
	stdoutOpt, stdout := process.WithStdout()
	stderrOpt, stderr := process.WithStderr()

	err := m.exec.Exec("git", []string{
		"ls-remote",
		"--heads",
		repoURL,
		"refs/heads/" + branch,
	}, stdoutOpt, stderrOpt)

	if err != nil {
		return false, fmt.Errorf("%w: git ls-remote %s: %s", ErrCommandExecutionFailed, repoURL, strings.TrimSpace(stderr.String()))
	}

	return lsRemoteHasBranch(stdout.String(), branch), nil
}

// lsRemoteHasBranch checks the output of 'git ls-remote' for an exact match of
// the branch, as ls-remote also matches refs that only end with the pattern.
func lsRemoteHasBranch(output string, branch string) bool {
	ref := "refs/heads/" + branch

	for line := range strings.Lines(output) {
		fields := strings.Fields(line)

		if len(fields) == 2 && fields[1] == ref {
			return true
		}
	}

	return false
}
