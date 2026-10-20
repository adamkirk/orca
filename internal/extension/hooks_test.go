package extension

import (
	"context"
	"errors"
	"testing"

	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hooksWorkspace has an api whose clone.post runs its own 'env', then the
// workspace's 'install-deps', and a workspace whose init.post runs 'seed'.
func hooksWorkspace() *workspace.Workspace {
	ws := trustWorkspace()
	ws.Hooks = map[workspace.Hook][]workspace.HookEntry{
		workspace.HookInitPost: {{Extension: "seed", Scope: workspace.HookScopeWorkspace}},
	}
	ws.Projects[0].Config.Extensions = append(ws.Projects[0].Config.Extensions, command("migrate", workspace.CommandSpec{Command: "./bin/migrate"}))
	ws.Projects[0].Config.Hooks = map[workspace.Hook][]workspace.HookEntry{
		workspace.HookClonePost: {
			{Extension: "migrate", Scope: workspace.HookScopeProject},
			{Extension: "install-deps", Scope: workspace.HookScopeWorkspace},
		},
	}

	return ws
}

func Test_RunHook(t *testing.T) {
	failure := errors.New("exit status 1")

	tests := []struct {
		name      string
		project   string
		cmd       RunHookCommand
		executor  stubExecutor
		answer    bool
		want      []ran
		wantLines []string
		wantAsked int
		wantErr   error
	}{
		{
			name:      "a project's hook, in order, all in its directory",
			project:   "api",
			cmd:       RunHookCommand{Hook: "clone.post"},
			answer:    true,
			wantAsked: 1,
			want: []ran{
				{name: "./bin/migrate", args: []string{}, dir: "/code/acme/api"},
				{name: "npm", args: []string{"ci"}, dir: "/code/acme/api"},
			},
			wantLines: []string{
				"This workspace wants to run the following on your machine:", "",
				"api clone.post: migrate",
				"api clone.post: install-deps",
			},
		},
		{
			name:    "--yes doesn't ask",
			project: "api",
			cmd:     RunHookCommand{Hook: "clone.post", Yes: true},
			want: []ran{
				{name: "./bin/migrate", args: []string{}, dir: "/code/acme/api"},
				{name: "npm", args: []string{"ci"}, dir: "/code/acme/api"},
			},
			wantLines: []string{"api clone.post: migrate", "api clone.post: install-deps"},
		},
		{
			name:      "a hook only the workspace has, from inside a project",
			project:   "api",
			cmd:       RunHookCommand{Hook: "init.post"},
			answer:    true,
			wantAsked: 1,
			want:      []ran{{name: "./scripts/seed.sh", args: []string{}, dir: "/code/acme/local-env"}},
			wantLines: []string{"This workspace wants to run the following on your machine:", "", "init.post: seed"},
		},
		{
			name:      "a hook with nothing in it",
			project:   "web",
			cmd:       RunHookCommand{Hook: "clone.post"},
			want:      []ran{},
			wantLines: []string{"web has no clone.post hook, so there's nothing to run."},
		},
		{
			name:      "declining runs nothing",
			project:   "api",
			cmd:       RunHookCommand{Hook: "clone.post"},
			wantAsked: 1,
			wantErr:   ErrTrustDeclined,
		},
		{
			name:    "a project's hook needs a project",
			cmd:     RunHookCommand{Hook: "clone.post"},
			wantErr: workspace.ErrUnknownHook,
		},
		{
			name:    "an unknown hook",
			project: "api",
			cmd:     RunHookCommand{Hook: "clone.pre"},
			wantErr: workspace.ErrUnknownHook,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := hooksWorkspace()
			execCtx := workspace.ExecutionContext{Workspace: ws}

			if tt.project != "" {
				p, err := ws.GetProject(tt.project)
				require.NoError(t, err)
				execCtx.Project = p
			}

			got := []ran{}
			out := &stubTui{answer: tt.answer}
			executor := tt.executor
			executor.ran = &got
			m := NewManager(afero.NewMemMapFs(), nil, executor, out, newTrustStore())

			err := m.RunHook(workspace.WithExecutionContext(context.Background(), execCtx), tt.cmd)

			assert.Equal(t, tt.wantAsked, out.asked)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantLines, out.lines)
		})
	}

	t.Run("a failing extension stops the hook, and says how to re-run it", func(t *testing.T) {
		ws := hooksWorkspace()
		got := []ran{}
		m := NewManager(afero.NewMemMapFs(), nil, stubExecutor{ran: &got, err: failure, failing: []string{"./bin/migrate"}}, &stubTui{}, newTrustStore())
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[0]})

		err := m.RunHook(ctx, RunHookCommand{Hook: "clone.post", Yes: true})

		assert.ErrorIs(t, err, failure)

		hookErr, ok := errors.AsType[HookFailedError](err)
		require.True(t, ok)
		assert.Equal(t, HookFailedError{Workspace: "acme", Project: "api", Hook: "clone.post", Extension: "migrate", Err: hookErr.Err}, hookErr)

		msg, _, _ := usererr.Message(err)
		assert.Equal(t, "Extension 'migrate' failed!", msg)
		assert.Equal(t, "orca ext -w acme -p api migrate", hookErr.RerunCommand())
		assert.Len(t, got, 1, "install-deps doesn't run")
	})

	t.Run("a failing workspace hook", func(t *testing.T) {
		ws := hooksWorkspace()
		got := []ran{}
		m := NewManager(afero.NewMemMapFs(), nil, stubExecutor{ran: &got, err: failure}, &stubTui{}, newTrustStore())
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws})

		err := m.RunHook(ctx, RunHookCommand{Hook: "init.post", Yes: true})

		hookErr, ok := errors.AsType[HookFailedError](err)
		require.True(t, ok)
		assert.Equal(t, "orca ext -w acme seed", hookErr.RerunCommand())
		assert.EqualError(t, hookErr, "the workspace's init.post hook failed running 'seed': Extension 'seed' failed!: exit status 1")
	})

	t.Run("a command's output is indented under the hook", func(t *testing.T) {
		ws := hooksWorkspace()
		got := []ran{}
		out := &stubTui{}
		m := NewManager(afero.NewMemMapFs(), nil, stubExecutor{ran: &got, output: "seeded\n"}, out, newTrustStore())
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws})

		require.NoError(t, m.RunHook(ctx, RunHookCommand{Hook: "init.post", Yes: true}))

		assert.Equal(t, "  seeded\n", out.output.String())
	})

	t.Run("a syncFile's message is indented under the hook", func(t *testing.T) {
		ws := hooksWorkspace()
		ws.Projects[0].Config.Hooks[workspace.HookClonePost] = []workspace.HookEntry{{Extension: "env", Scope: workspace.HookScopeProject}}
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/code/acme/api/.env.example", []byte("A=1\n"), 0644))
		out := &stubTui{}
		m := NewManager(fs, nil, nil, out, newTrustStore())
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[0]})

		require.NoError(t, m.RunHook(ctx, RunHookCommand{Hook: "clone.post", Yes: true}))

		assert.Equal(t, []string{"api clone.post: env", "  Created .env from .env.example."}, out.lines)
	})
}

func Test_HookChoices(t *testing.T) {
	ws := hooksWorkspace()
	ws.Hooks[workspace.HookUpPre] = []workspace.HookEntry{{Extension: "install-deps", Scope: workspace.HookScopeWorkspace}}
	m := NewManager(nil, nil, nil, nil, nil)

	t.Run("in a project, its hooks and the workspace's init.post", func(t *testing.T) {
		got, err := m.HookChoices(workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[0]}), HookChoicesQuery{})

		require.NoError(t, err)
		assert.Equal(t, []workspace.Choice{
			{Value: "clone.post", Description: "Runs migrate, install-deps"},
			{Value: "init.post", Description: "Runs seed"},
		}, got)
	})

	t.Run("outside of a project, the workspace's", func(t *testing.T) {
		got, err := m.HookChoices(workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws}), HookChoicesQuery{})

		require.NoError(t, err)
		assert.Equal(t, []workspace.Choice{
			{Value: "init.post", Description: "Runs seed"},
			{Value: "up.pre", Description: "Runs install-deps"},
		}, got)
	})
}
