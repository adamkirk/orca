package stack

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/adamkirk/orca/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingCompose records the projects started and stopped, and knows which
// are running. Anything else panics.
type recordingCompose struct {
	compose

	running map[string]bool
	started []string
	stopped []string

	// Shared with recordingHooks, when set, to check the order of everything
	log *[]string
}

func (c *recordingCompose) Up(_ *workspace.Workspace, p *workspace.Project) error {
	c.started = append(c.started, p.Name)

	if c.log != nil {
		*c.log = append(*c.log, "start "+p.Name)
	}

	return nil
}

func (c *recordingCompose) Down(_ *workspace.Workspace, p *workspace.Project) error {
	c.stopped = append(c.stopped, p.Name)

	if c.log != nil {
		*c.log = append(*c.log, "stop "+p.Name)
	}

	return nil
}

// recordingHooks records the hooks confirmed, and those run, as "api up.pre".
// It fails the hook named in failing, with err.
type recordingHooks struct {
	confirmed []workspace.HookRun
	log       *[]string

	err     error
	failing string
}

func hookName(run workspace.HookRun) string {
	return run.Project.Name + " " + run.Hook.String()
}

func (h *recordingHooks) ConfirmHooks(_ *workspace.Workspace, runs []workspace.HookRun, _ bool) error {
	h.confirmed = runs

	if h.failing == "confirm" {
		return h.err
	}

	return nil
}

func (h *recordingHooks) RunConfirmedHook(_ *workspace.Workspace, run workspace.HookRun) error {
	if h.log != nil {
		*h.log = append(*h.log, hookName(run))
	}

	if hookName(run) == h.failing {
		return h.err
	}

	return nil
}

func (c *recordingCompose) IsProjectRunning(_ *workspace.Workspace, p *workspace.Project) (bool, error) {
	return c.running[p.Name], nil
}

type recordingTUI struct {
	lines []string
}

func (r *recordingTUI) Info(msg ...string) { r.lines = append(r.lines, msg...) }

func newTestWorkspace() *workspace.Workspace {
	return &workspace.Workspace{Projects: []workspace.Project{
		{Name: "local-env", Disabled: true},
		{Name: "api", Requires: []string{"local-env"}},
		{Name: "web", Requires: []string{"api"}, Disabled: true},
		{Name: "admin", Disabled: true},
	}}
}

func TestUp(t *testing.T) {
	t.Run("starts enabled projects and their requirements", func(t *testing.T) {
		compose := &recordingCompose{}
		tui := &recordingTUI{}
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newTestWorkspace()})

		require.NoError(t, NewManager(compose, tui, &recordingHooks{}).Up(ctx, UpCommand{}))

		assert.Equal(t, []string{"local-env", "api"}, compose.started)
		assert.Equal(t, []string{"Warning: api requires local-env, which is disabled, so it's being started anyway."}, tui.lines)
	})

	t.Run("starts a disabled project when asked for it", func(t *testing.T) {
		compose := &recordingCompose{}
		tui := &recordingTUI{}
		ws := newTestWorkspace()
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[2]})

		require.NoError(t, NewManager(compose, tui, &recordingHooks{}).Up(ctx, UpCommand{}))

		assert.Equal(t, []string{"web"}, compose.started)
		assert.Equal(t, []string{"Note: web is disabled, but starting it as you asked for it."}, tui.lines)
	})
}

func newLibraryWorkspace() *workspace.Workspace {
	return &workspace.Workspace{Projects: []workspace.Project{
		{Name: "sdk", Type: workspace.ProjectTypeLibrary, IsRegistered: true},
		{Name: "tools", Type: workspace.ProjectTypeLibrary},
		{Name: "api"},
		{Name: "web", Requires: []string{"sdk", "tools", "api"}},
	}}
}

func TestUpLibraries(t *testing.T) {
	t.Run("skips libraries, and warns about required ones that aren't cloned", func(t *testing.T) {
		compose := &recordingCompose{}
		tui := &recordingTUI{}
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newLibraryWorkspace()})

		require.NoError(t, NewManager(compose, tui, &recordingHooks{}).Up(ctx, UpCommand{}))

		assert.Equal(t, []string{"api", "web"}, compose.started)
		assert.Equal(t, []string{"Warning: web requires tools, which isn't cloned; clone it with 'orca ws clone -p tools'."}, tui.lines)
	})

	t.Run("warns when starting one project", func(t *testing.T) {
		compose := &recordingCompose{}
		tui := &recordingTUI{}
		ws := newLibraryWorkspace()
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[3]})

		require.NoError(t, NewManager(compose, tui, &recordingHooks{}).Up(ctx, UpCommand{}))

		assert.Equal(t, []string{"web"}, compose.started)
		assert.Equal(t, []string{"Warning: web requires tools, which isn't cloned; clone it with 'orca ws clone -p tools'."}, tui.lines)
	})

	t.Run("down skips libraries", func(t *testing.T) {
		compose := &recordingCompose{running: map[string]bool{"sdk": true}}
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newLibraryWorkspace()})

		require.NoError(t, NewManager(compose, &recordingTUI{}, &recordingHooks{}).Down(ctx, DownCommand{}))

		assert.Equal(t, []string{"web", "api"}, compose.stopped)
	})
}

func TestDown(t *testing.T) {
	t.Run("stops what up starts, and disabled projects still running", func(t *testing.T) {
		compose := &recordingCompose{running: map[string]bool{"web": true}}
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newTestWorkspace()})

		require.NoError(t, NewManager(compose, &recordingTUI{}, &recordingHooks{}).Down(ctx, DownCommand{}))

		assert.Equal(t, []string{"web", "api", "local-env"}, compose.stopped)
	})

	t.Run("stops a disabled project when asked for it", func(t *testing.T) {
		compose := &recordingCompose{}
		ws := newTestWorkspace()
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[3]})

		require.NoError(t, NewManager(compose, &recordingTUI{}, &recordingHooks{}).Down(ctx, DownCommand{}))

		assert.Equal(t, []string{"admin"}, compose.stopped)
	})
}

func TestUpDownHooks(t *testing.T) {
	newManager := func(failing string) (*Manager, *recordingHooks, *[]string) {
		log := []string{}
		hooks := &recordingHooks{log: &log, failing: failing, err: errors.New("hook failed")}

		return NewManager(&recordingCompose{log: &log}, &recordingTUI{}, hooks), hooks, &log
	}

	t.Run("a workspace-wide up runs each project's own hooks around its start", func(t *testing.T) {
		m, hooks, log := newManager("")
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newTestWorkspace()})

		require.NoError(t, m.Up(ctx, UpCommand{}))

		assert.Equal(t, []string{
			"local-env up.pre", "start local-env", "local-env up.post",
			"api up.pre", "start api", "api up.post",
		}, *log)

		confirmed := []string{}

		for _, run := range hooks.confirmed {
			confirmed = append(confirmed, hookName(run))
		}

		assert.Equal(t, []string{"local-env up.pre", "local-env up.post", "api up.pre", "api up.post"}, confirmed, "everything is asked about first")
	})

	t.Run("a workspace-wide down runs in reverse", func(t *testing.T) {
		m, _, log := newManager("")
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newTestWorkspace()})

		require.NoError(t, m.Down(ctx, DownCommand{}))

		assert.Equal(t, []string{
			"api down.pre", "stop api", "api down.post",
			"local-env down.pre", "stop local-env", "local-env down.post",
		}, *log)
	})

	t.Run("one project only runs its own", func(t *testing.T) {
		m, _, log := newManager("")
		ws := newTestWorkspace()
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[1]})

		require.NoError(t, m.Up(ctx, UpCommand{}))

		assert.Equal(t, []string{"api up.pre", "start api", "api up.post"}, *log)
	})

	t.Run("disabled projects that aren't started don't run theirs", func(t *testing.T) {
		m, _, log := newManager("")
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newTestWorkspace()})

		require.NoError(t, m.Up(ctx, UpCommand{}))

		assert.NotContains(t, strings.Join(*log, "\n"), "web")
		assert.NotContains(t, strings.Join(*log, "\n"), "admin")
	})

	t.Run("declining starts nothing", func(t *testing.T) {
		m, _, log := newManager("confirm")
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newTestWorkspace()})

		assert.Error(t, m.Up(ctx, UpCommand{}))
		assert.Empty(t, *log)
	})

	t.Run("a failing pre hook stops before the project starts", func(t *testing.T) {
		m, _, log := newManager("api up.pre")
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newTestWorkspace()})

		assert.Error(t, m.Up(ctx, UpCommand{}))
		assert.Equal(t, []string{"local-env up.pre", "start local-env", "local-env up.post", "api up.pre"}, *log)
	})

	t.Run("a failing post hook fails, after the project's started", func(t *testing.T) {
		m, _, log := newManager("api up.post")
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newTestWorkspace()})

		assert.Error(t, m.Up(ctx, UpCommand{}))
		assert.Equal(t, []string{"api up.pre", "start api", "api up.post"}, (*log)[3:])
	})
}
