package workspace_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubResolver struct {
	execCtx workspace.ExecutionContext
	err     error

	calls [][2]string
}

func (r *stubResolver) Resolve(ws string, project string) (workspace.ExecutionContext, error) {
	r.calls = append(r.calls, [2]string{ws, project})

	return r.execCtx, r.err
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

func newMiddleware(resolver *stubResolver) *workspace.ExecutionContextMiddleware {
	ec := workspace.NewExecutionContextMiddleware(resolver)

	ec.Target(func(cmd targetedCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{
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

	resolved := workspace.ExecutionContext{
		Workspace:        &workspace.Workspace{Name: "acme"},
		WorkingDirectory: "/somewhere",
	}
	noWorkspace := fmt.Errorf("%w: nope", workspace.ErrCouldNotDetermineWorkspace)
	otherErr := fmt.Errorf("%w: missing", workspace.ErrUnknownWorkspace)

	tests := []struct {
		name       string
		cmd        targetedCommand
		resolveCtx workspace.ExecutionContext
		resolveErr error
		expectCtx  workspace.ExecutionContext
		expectErr  error
		expectMsg  string
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
			expectCtx:  workspace.ExecutionContext{WorkingDirectory: wd},
		},
		{
			name:       "requires a workspace when it isn't optional",
			cmd:        targetedCommand{All: true},
			resolveErr: noWorkspace,
			expectErr:  noWorkspace,
			expectMsg:  noWorkspaceMsg,
		},
		{
			name:       "requires a workspace for a specific project",
			cmd:        targetedCommand{Project: "api"},
			resolveErr: noWorkspace,
			expectErr:  noWorkspace,
			expectMsg:  noWorkspaceMsg,
		},
		{
			name:       "other errors are returned",
			resolveErr: otherErr,
			expectErr:  otherErr,
			expectMsg:  "Failed to determine the workspace or project!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &stubResolver{execCtx: tt.resolveCtx, err: tt.resolveErr}
			b := bus.New()
			b.Use(newMiddleware(resolver).Handle)

			handled := false
			var hookCtx workspace.ExecutionContext

			b.Before(func(ctx context.Context, cmd targetedCommand) error {
				hookCtx, _ = workspace.ExecutionContextFrom(ctx)

				return nil
			})
			b.HandleVoid(func(ctx context.Context, cmd targetedCommand) error {
				handled = true

				execCtx, err := workspace.ExecutionContextFrom(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.expectCtx, execCtx)

				return nil
			})

			_, err := b.Dispatch(context.Background(), tt.cmd)

			assert.Equal(t, [][2]string{{tt.cmd.Workspace, tt.cmd.Project}}, resolver.calls)
			msg, _, _ := usererr.Message(err)
			assert.Equal(t, tt.expectMsg, msg)

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

type servicesOnlyCommand struct {
	bus.Returns[bus.Void]
}

func (servicesOnlyCommand) MessageName() string { return "services-only" }

func Test_ExecutionContextMiddleware_ServicesOnly(t *testing.T) {
	ws := &workspace.Workspace{
		Name: "acme",
		Projects: []workspace.Project{
			{Name: "api", Type: workspace.ProjectTypeService},
			{Name: "sdk", Type: workspace.ProjectTypeLibrary},
		},
	}

	tests := []struct {
		name      string
		project   *workspace.Project
		expectErr bool
	}{
		{name: "a service is handled", project: &ws.Projects[0]},
		{name: "the whole workspace is handled"},
		{name: "a library fails", project: &ws.Projects[1], expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ec := workspace.NewExecutionContextMiddleware(&stubResolver{
				execCtx: workspace.ExecutionContext{Workspace: ws, Project: tt.project},
			})
			ec.Target(func(servicesOnlyCommand) workspace.ExecutionTarget {
				return workspace.ExecutionTarget{ServicesOnly: true}
			})

			b := bus.New()
			b.Use(ec.Handle)

			handled := false

			b.HandleVoid(func(context.Context, servicesOnlyCommand) error {
				handled = true

				return nil
			})

			_, err := b.Dispatch(context.Background(), servicesOnlyCommand{})

			if !tt.expectErr {
				require.NoError(t, err)
				assert.True(t, handled)
				return
			}

			assert.ErrorIs(t, err, workspace.ErrLibraryProject)
			msg, _, _ := usererr.Message(err)
			assert.Equal(t, "sdk is a library project, so it has no compose stack for this command to act on!", msg)
			assert.False(t, handled)
		})
	}
}

func Test_ExecutionContextMiddleware_UntargetedPassesThrough(t *testing.T) {
	resolver := &stubResolver{}

	b := bus.New()
	b.Use(newMiddleware(resolver).Handle)

	var handlerErr error

	b.HandleVoid(func(ctx context.Context, cmd untargetedCommand) error {
		_, handlerErr = workspace.ExecutionContextFrom(ctx)

		return nil
	})

	_, err := b.Dispatch(context.Background(), untargetedCommand{})

	require.NoError(t, err)
	assert.Empty(t, resolver.calls)
	assert.Equal(t, workspace.ErrNoExecutionContext, handlerErr)
}

func Test_ExecutionContextMiddleware_Target(t *testing.T) {
	ec := newMiddleware(&stubResolver{})

	assert.Equal(t, []string{"workspace_test.targetedCommand"}, ec.TargetedTypes())

	assert.Panics(t, func() {
		ec.Target(func(cmd targetedCommand) workspace.ExecutionTarget {
			return workspace.ExecutionTarget{}
		})
	})
}

func Test_ExecutionContextFrom_Missing(t *testing.T) {
	_, err := workspace.ExecutionContextFrom(context.Background())

	assert.Equal(t, workspace.ErrNoExecutionContext, err)
}
