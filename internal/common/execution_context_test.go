package common_test

import (
	"context"
	"os"
	"testing"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubResolver struct {
	execCtx common.ExecutionContext
	err     error

	calls [][2]string
}

func (r *stubResolver) Resolve(ws string, project string) (common.ExecutionContext, error) {
	r.calls = append(r.calls, [2]string{ws, project})

	return r.execCtx, r.err
}

type stubTui struct {
	recorded []string
}

func (t *stubTui) RecordIfError(msg string, err error) error {
	if err != nil {
		t.recorded = append(t.recorded, msg)
	}

	return err
}

type targetedCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
	All       bool
}

func (targetedCommand) MessageName() string { return "targeted" }

type untargetedCommand struct {
	bus.Returns[bus.Void]
}

func (untargetedCommand) MessageName() string { return "untargeted" }

const noWorkspaceMsg = "Could not determine the workspace; run this from within a project, pass --workspace, or choose one with 'orca ws switch'."

func newMiddleware(resolver *stubResolver, tui *stubTui) *common.ExecutionContextMiddleware {
	ec := common.NewExecutionContextMiddleware(resolver, tui)

	ec.Target(func(cmd targetedCommand) common.ExecutionTarget {
		return common.ExecutionTarget{
			Workspace:         cmd.Workspace,
			Project:           cmd.Project,
			WorkspaceOptional: !cmd.All,
		}
	})

	return ec
}

func Test_ExecutionContextMiddleware_Resolve(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	resolved := common.ExecutionContext{
		Workspace:        &common.Workspace{Name: "acme"},
		WorkingDirectory: "/somewhere",
	}
	noWorkspace := common.ErrCouldNotDetermineWorkspace{Message: "nope"}
	otherErr := common.ErrUnknownWorkspace{Name: "missing"}

	tests := []struct {
		name           string
		cmd            targetedCommand
		resolveCtx     common.ExecutionContext
		resolveErr     error
		expectCtx      common.ExecutionContext
		expectErr      error
		expectRecorded []string
	}{
		{
			name:       "resolved context is passed to the handler",
			cmd:        targetedCommand{Workspace: "acme", Project: "api"},
			resolveCtx: resolved,
			expectCtx:  resolved,
		},
		{
			name:       "falls back to the working directory when the workspace is optional",
			resolveErr: noWorkspace,
			expectCtx:  common.ExecutionContext{WorkingDirectory: wd},
		},
		{
			name:           "requires a workspace when it isn't optional",
			cmd:            targetedCommand{All: true},
			resolveErr:     noWorkspace,
			expectErr:      noWorkspace,
			expectRecorded: []string{noWorkspaceMsg},
		},
		{
			name:           "requires a workspace for a specific project",
			cmd:            targetedCommand{Project: "api"},
			resolveErr:     noWorkspace,
			expectErr:      noWorkspace,
			expectRecorded: []string{noWorkspaceMsg},
		},
		{
			name:           "other errors are returned",
			resolveErr:     otherErr,
			expectErr:      otherErr,
			expectRecorded: []string{"Failed to determine the workspace or project!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &stubResolver{execCtx: tt.resolveCtx, err: tt.resolveErr}
			tui := &stubTui{}

			b := bus.New()
			b.Use(newMiddleware(resolver, tui).Handle)

			handled := false
			var hookCtx common.ExecutionContext

			b.Before(func(ctx context.Context, cmd targetedCommand) error {
				hookCtx, _ = common.ExecutionContextFrom(ctx)

				return nil
			})
			b.HandleVoid(func(ctx context.Context, cmd targetedCommand) error {
				handled = true

				execCtx, err := common.ExecutionContextFrom(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.expectCtx, execCtx)

				return nil
			})

			_, err := b.Dispatch(context.Background(), tt.cmd)

			assert.Equal(t, [][2]string{{tt.cmd.Workspace, tt.cmd.Project}}, resolver.calls)
			assert.Equal(t, tt.expectRecorded, tui.recorded)

			if tt.expectErr != nil {
				assert.ErrorIs(t, err, tt.expectErr)
				assert.False(t, handled)
				return
			}

			require.NoError(t, err)
			assert.True(t, handled)

			// Hooks run inside the middleware, so see it too
			assert.Equal(t, tt.expectCtx, hookCtx)
		})
	}
}

func Test_ExecutionContextMiddleware_UntargetedPassesThrough(t *testing.T) {
	resolver := &stubResolver{}

	b := bus.New()
	b.Use(newMiddleware(resolver, &stubTui{}).Handle)

	var handlerErr error

	b.HandleVoid(func(ctx context.Context, cmd untargetedCommand) error {
		_, handlerErr = common.ExecutionContextFrom(ctx)

		return nil
	})

	_, err := b.Dispatch(context.Background(), untargetedCommand{})

	require.NoError(t, err)
	assert.Empty(t, resolver.calls)
	assert.Equal(t, common.ErrNoExecutionContext{}, handlerErr)
}

func Test_ExecutionContextMiddleware_Target(t *testing.T) {
	ec := newMiddleware(&stubResolver{}, &stubTui{})

	assert.Equal(t, []string{"common_test.targetedCommand"}, ec.TargetedTypes())

	assert.Panics(t, func() {
		ec.Target(func(cmd targetedCommand) common.ExecutionTarget {
			return common.ExecutionTarget{}
		})
	})
}

func Test_ExecutionContextFrom_Missing(t *testing.T) {
	_, err := common.ExecutionContextFrom(context.Background())

	assert.Equal(t, common.ErrNoExecutionContext{}, err)
}
