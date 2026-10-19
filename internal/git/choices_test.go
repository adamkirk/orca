package git

import (
	"context"
	"testing"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchChoices(t *testing.T) {
	m := NewManager(process.NewExecutor(), stubTUI{t})

	t.Run("local branches, then remote-only ones, without the current branch", func(t *testing.T) {
		clone := cloneWithRemoteBranch(t)
		git(t, clone, "branch", "feature/local")
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{WorkingDirectory: clone})

		got, err := m.BranchChoices(ctx, BranchChoicesQuery{})

		require.NoError(t, err)
		assert.Equal(t, []workspace.Choice{
			{Value: "feature/local"},
			{Value: "feature/remote-only"},
		}, got)
	})

	t.Run("outside a git repository", func(t *testing.T) {
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{WorkingDirectory: t.TempDir()})

		_, err := m.BranchChoices(ctx, BranchChoicesQuery{})

		assert.ErrorIs(t, err, workspace.ErrInvalidExecutionContext)
	})
}
