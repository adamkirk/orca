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

// hooksWorkspace is trustWorkspace, where api's clone.post runs its own
// 'migrate' then 'install-deps', and the workspace's init.post runs
// local-env's 'seed'.
func hooksWorkspace() *workspace.Workspace {
	ws := trustWorkspace()
	ws.InitHooks = []workspace.InitHookEntry{{Project: "local-env", Extension: "seed"}}

	api := &ws.Projects[1]
	api.Config.Extensions = append(api.Config.Extensions, command("migrate", workspace.CommandSpec{Command: "./bin/migrate"}))
	api.Config.Hooks = map[workspace.Hook][]workspace.HookEntry{
		workspace.HookClonePost: {{Extension: "migrate"}, {Extension: "install-deps"}},
	}

	return ws
}

func Test_RunHook(t *testing.T) {
	tests := []struct {
		name      string
		project   string
		cmd       RunHookCommand
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
				"", "This workspace wants to run the following on your machine:", "",
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
			name:      "the workspace's init.post, from inside another project, runs in its own project",
			project:   "api",
			cmd:       RunHookCommand{Hook: "init.post", Yes: true},
			want:      []ran{{name: "./scripts/seed.sh", args: []string{}, dir: "/code/acme/local-env"}},
			wantLines: []string{"local-env init.post: seed"},
		},
		{
			name:      "the workspace's init.post, outside of any project",
			cmd:       RunHookCommand{Hook: "init.post", Yes: true},
			want:      []ran{{name: "./scripts/seed.sh", args: []string{}, dir: "/code/acme/local-env"}},
			wantLines: []string{"local-env init.post: seed"},
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
			m := NewManager(afero.NewMemMapFs(), nil, stubExecutor{ran: &got}, out, newTrustStore())

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

	failure := errors.New("exit status 1")

	t.Run("a failing extension stops the hook, and says how to re-run it", func(t *testing.T) {
		ws := hooksWorkspace()
		got := []ran{}
		m := NewManager(afero.NewMemMapFs(), nil, stubExecutor{ran: &got, err: failure, failing: []string{"./bin/migrate"}}, &stubTui{}, newTrustStore())
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[1]})

		err := m.RunHook(ctx, RunHookCommand{Hook: "clone.post", Yes: true})

		assert.ErrorIs(t, err, failure)

		hookErr, ok := errors.AsType[HookFailedError](err)
		require.True(t, ok)
		assert.Equal(t, HookFailedError{Workspace: "acme", Project: "api", Hook: "clone.post", Extension: "migrate", Err: hookErr.Err}, hookErr)

		msg, _, _ := usererr.Message(err)
		assert.Equal(t, "Extension 'migrate' failed!", msg)
		assert.Equal(t, "'migrate', in api's clone.post hook", hookErr.Describe())
		assert.Equal(t, "orca ext -w acme -p api migrate", hookErr.RerunCommand())
		assert.Equal(t, "orca hooks run -w acme -p api clone.post", hookErr.RerunHookCommand())
		assert.Len(t, got, 1, "install-deps doesn't run")
	})

	t.Run("a failing init.post names its project, and the workspace's hook", func(t *testing.T) {
		ws := hooksWorkspace()
		got := []ran{}
		m := NewManager(afero.NewMemMapFs(), nil, stubExecutor{ran: &got, err: failure}, &stubTui{}, newTrustStore())
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws})

		err := m.RunHook(ctx, RunHookCommand{Hook: "init.post", Yes: true})

		hookErr, ok := errors.AsType[HookFailedError](err)
		require.True(t, ok)
		assert.Equal(t, "local-env's 'seed', in the workspace's init.post hook", hookErr.Describe())
		assert.Equal(t, "orca ext -w acme -p local-env seed", hookErr.RerunCommand())
		assert.Equal(t, "orca hooks run -w acme init.post", hookErr.RerunHookCommand())
	})

	t.Run("init.post naming a project that isn't cloned", func(t *testing.T) {
		ws := hooksWorkspace()
		ws.Projects[0].IsRegistered = false
		got := []ran{}
		m := NewManager(afero.NewMemMapFs(), nil, stubExecutor{ran: &got}, &stubTui{}, newTrustStore())

		err := m.RunHook(workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws}), RunHookCommand{Hook: "init.post", Yes: true})

		assert.ErrorIs(t, err, ErrProjectNotCloned)
		assert.Empty(t, got)
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
		ws.Projects[1].Config.Hooks[workspace.HookClonePost] = []workspace.HookEntry{{Extension: "env"}}
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/code/acme/api/.env.example", []byte("A=1\n"), 0644))
		out := &stubTui{}
		m := NewManager(fs, nil, nil, out, newTrustStore())
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[1]})

		require.NoError(t, m.RunHook(ctx, RunHookCommand{Hook: "clone.post", Yes: true}))

		assert.Equal(t, []string{"api clone.post: env", "  Created .env from .env.example."}, out.lines)
	})
}

func Test_HookChoices(t *testing.T) {
	ws := hooksWorkspace()
	m := NewManager(nil, nil, nil, nil, nil)

	t.Run("in a project, its own, and the workspace's init.post", func(t *testing.T) {
		got, err := m.HookChoices(workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[1]}), HookChoicesQuery{})

		require.NoError(t, err)
		assert.Equal(t, []workspace.Choice{
			{Value: "clone.post", Description: "Runs migrate, install-deps"},
			{Value: "init.post", Description: "Runs local-env's seed"},
		}, got)
	})

	t.Run("outside of a project, the workspace's init.post", func(t *testing.T) {
		got, err := m.HookChoices(workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws}), HookChoicesQuery{})

		require.NoError(t, err)
		assert.Equal(t, []workspace.Choice{{Value: "init.post", Description: "Runs local-env's seed"}}, got)
	})
}

func Test_SetUp(t *testing.T) {
	newManager := func(out *stubTui, got *[]ran) *Manager {
		return NewManager(afero.NewMemMapFs(), stubCompose{ran: got, services: []string{"app"}}, stubExecutor{ran: got}, out, newTrustStore())
	}

	t.Run("each project's clone.post, then init.post, asked about together", func(t *testing.T) {
		ws := hooksWorkspace()
		ws.Projects[2].Config.Hooks = map[workspace.Hook][]workspace.HookEntry{
			workspace.HookClonePost: {{Extension: "install-deps"}},
		}
		got := []ran{}
		out := &stubTui{answer: true}

		require.NoError(t, newManager(out, &got).SetUp(ws, workspace.SetUpOpts{Projects: []string{"local-env", "web", "api"}, Initialise: true}))

		assert.Equal(t, 1, out.asked)
		assert.Equal(t, []ran{
			{name: "npm", args: []string{"ci"}, dir: "/code/acme/web"},
			{name: "./bin/migrate", args: []string{}, dir: "/code/acme/api"},
			{name: "npm", args: []string{"ci"}, dir: "/code/acme/api"},
			{name: "./scripts/seed.sh", args: []string{}, dir: "/code/acme/local-env"},
		}, got)
		assert.Equal(t, [][][]string{{
			{"web", "clone.post", "install-deps", "command", "npm ci"},
			{"api", "clone.post", "migrate", "command", "./bin/migrate"},
			{"api", "clone.post", "install-deps", "command", "npm ci"},
			{"local-env", "init.post", "seed", "command", "./scripts/seed.sh"},
		}}, out.tables)
	})

	t.Run("without init.post", func(t *testing.T) {
		got := []ran{}

		require.NoError(t, newManager(&stubTui{}, &got).SetUp(hooksWorkspace(), workspace.SetUpOpts{Projects: []string{"api"}, Yes: true}))

		assert.Len(t, got, 2)
	})

	t.Run("nothing to run asks nothing", func(t *testing.T) {
		got := []ran{}
		out := &stubTui{}

		require.NoError(t, newManager(out, &got).SetUp(hooksWorkspace(), workspace.SetUpOpts{Projects: []string{"web"}}))

		assert.Zero(t, out.asked)
		assert.Empty(t, got)
	})

	t.Run("declining runs nothing", func(t *testing.T) {
		got := []ran{}

		err := newManager(&stubTui{}, &got).SetUp(hooksWorkspace(), workspace.SetUpOpts{Projects: []string{"api"}, Initialise: true})

		assert.ErrorIs(t, err, ErrTrustDeclined)
		assert.Empty(t, got)
	})

	t.Run("a service command from a hook doesn't use the terminal", func(t *testing.T) {
		ws := hooksWorkspace()
		api := &ws.Projects[1]
		api.Config.Extensions = append(api.Config.Extensions, command("composer", workspace.CommandSpec{Command: "composer install", Service: "app"}))
		api.Config.Hooks[workspace.HookClonePost] = []workspace.HookEntry{{Extension: "composer"}}
		got := []ran{}

		require.NoError(t, newManager(&stubTui{}, &got).SetUp(ws, workspace.SetUpOpts{Projects: []string{"api"}, Yes: true}))

		assert.Equal(t, []ran{{name: "composer", args: []string{"install"}, dir: "/code/acme/api", service: "app", captured: true}}, got)
	})
}

func Test_ConfirmHooks_RunConfirmedHook(t *testing.T) {
	ws := hooksWorkspace()
	root, api := &ws.Projects[0], &ws.Projects[1]
	root.Config.Hooks = map[workspace.Hook][]workspace.HookEntry{workspace.HookUpPre: {{Extension: "seed"}}}
	api.Config.Hooks[workspace.HookUpPost] = []workspace.HookEntry{{Extension: "migrate"}}
	got := []ran{}
	out := &stubTui{answer: true}
	m := NewManager(afero.NewMemMapFs(), nil, stubExecutor{ran: &got}, out, newTrustStore())
	runs := []workspace.HookRun{
		{Project: root, Hook: workspace.HookUpPre},
		{Project: root, Hook: workspace.HookUpPost},
		{Project: api, Hook: workspace.HookUpPre},
		{Project: api, Hook: workspace.HookUpPost},
	}

	require.NoError(t, m.ConfirmHooks(ws, runs, false))

	assert.Equal(t, 1, out.asked)
	assert.Equal(t, [][][]string{{
		{"local-env", "up.pre", "seed", "command", "./scripts/seed.sh"},
		{"api", "up.post", "migrate", "command", "./bin/migrate"},
	}}, out.tables)

	for _, run := range runs {
		require.NoError(t, m.RunConfirmedHook(ws, run))
	}

	assert.Equal(t, 1, out.asked, "running them asks nothing more")
	assert.Equal(t, []ran{
		{name: "./scripts/seed.sh", args: []string{}, dir: "/code/acme/local-env"},
		{name: "./bin/migrate", args: []string{}, dir: "/code/acme/api"},
	}, got)
}
