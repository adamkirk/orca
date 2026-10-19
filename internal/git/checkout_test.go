package git

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTUI fails the test if asked to choose, as these tests expect a single
// match.
type stubTUI struct {
	t *testing.T
}

func (s stubTUI) Info(msg ...string)                     {}
func (s stubTUI) Success(msg ...string)                  {}
func (s stubTUI) NewLine()                               {}
func (s stubTUI) Table(header []string, rows [][]string) {}

func (s stubTUI) PresentChoices(opts []string, title string) (string, error) {
	s.t.Fatalf("unexpected choice between %v", opts)

	return "", nil
}

// git runs git in dir, failing the test if it fails, and returns its output.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=orca", "GIT_AUTHOR_EMAIL=orca@example.com",
		"GIT_COMMITTER_NAME=orca", "GIT_COMMITTER_EMAIL=orca@example.com",
	)

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)

	return strings.TrimSpace(string(out))
}

// cloneWithRemoteBranch returns a clone of a repository with a main branch,
// and a feature/remote-only branch that only exists on its origin.
func cloneWithRemoteBranch(t *testing.T) string {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git isn't installed")
	}

	root := t.TempDir()
	origin := filepath.Join(root, "origin")
	clone := filepath.Join(root, "clone")

	git(t, root, "init", "--initial-branch=main", origin)
	git(t, origin, "commit", "--allow-empty", "-m", "initial")
	git(t, origin, "branch", "feature/remote-only")
	git(t, root, "clone", origin, clone)

	return clone
}

func TestCheckoutRemoteOnlyBranch(t *testing.T) {
	tests := []struct {
		name string
		cmd  CheckoutCommand
	}{
		{name: "by its full name", cmd: CheckoutCommand{Name: "feature/remote-only"}},
		{name: "by part of its name", cmd: CheckoutCommand{Name: "remote-only"}},
		{name: "with --create, it's checked out rather than created", cmd: CheckoutCommand{Name: "feature/remote-only", Create: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clone := cloneWithRemoteBranch(t)
			m := NewManager(process.NewExecutor(), stubTUI{t})
			ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{WorkingDirectory: clone})

			require.NoError(t, m.Checkout(ctx, tt.cmd))

			assert.Equal(t, "feature/remote-only", git(t, clone, "branch", "--show-current"))
			assert.Equal(t, "origin/feature/remote-only", git(t, clone, "rev-parse", "--abbrev-ref", "@{upstream}"), "tracks the remote branch")
		})
	}
}

func TestCheckoutPrefersLocalBranches(t *testing.T) {
	clone := cloneWithRemoteBranch(t)
	git(t, clone, "branch", "feature/local")

	m := NewManager(process.NewExecutor(), stubTUI{t})
	ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{WorkingDirectory: clone})

	branches, err := m.searchBranches(searchBranchesOpts{Dir: clone, IncludeRemote: true})

	require.NoError(t, err)
	assert.Equal(t, Branches{
		{Name: "feature/local"},
		{Name: "main", Current: true},
		{Name: "feature/remote-only", RemoteOnly: true},
	}, branches, "local branches first, then remote-only ones, without their prefix")

	require.NoError(t, m.Checkout(ctx, CheckoutCommand{Name: "feature/local"}))
	assert.Equal(t, "feature/local", git(t, clone, "branch", "--show-current"))
}
