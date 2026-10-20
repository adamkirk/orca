package stack

import (
	"context"
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
}

func (c *recordingCompose) Up(_ *workspace.Workspace, p *workspace.Project) error {
	c.started = append(c.started, p.Name)

	return nil
}

func (c *recordingCompose) Down(_ *workspace.Workspace, p *workspace.Project) error {
	c.stopped = append(c.stopped, p.Name)

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

		require.NoError(t, NewManager(compose, tui).Up(ctx, UpCommand{}))

		assert.Equal(t, []string{"local-env", "api"}, compose.started)
		assert.Equal(t, []string{"Warning: api requires local-env, which is disabled, so it's being started anyway."}, tui.lines)
	})

	t.Run("starts a disabled project when asked for it", func(t *testing.T) {
		compose := &recordingCompose{}
		tui := &recordingTUI{}
		ws := newTestWorkspace()
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[2]})

		require.NoError(t, NewManager(compose, tui).Up(ctx, UpCommand{}))

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

		require.NoError(t, NewManager(compose, tui).Up(ctx, UpCommand{}))

		assert.Equal(t, []string{"api", "web"}, compose.started)
		assert.Equal(t, []string{"Warning: web requires tools, which isn't cloned; clone it with 'orca ws clone -p tools'."}, tui.lines)
	})

	t.Run("warns when starting one project", func(t *testing.T) {
		compose := &recordingCompose{}
		tui := &recordingTUI{}
		ws := newLibraryWorkspace()
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[3]})

		require.NoError(t, NewManager(compose, tui).Up(ctx, UpCommand{}))

		assert.Equal(t, []string{"web"}, compose.started)
		assert.Equal(t, []string{"Warning: web requires tools, which isn't cloned; clone it with 'orca ws clone -p tools'."}, tui.lines)
	})

	t.Run("down skips libraries", func(t *testing.T) {
		compose := &recordingCompose{running: map[string]bool{"sdk": true}}
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newLibraryWorkspace()})

		require.NoError(t, NewManager(compose, &recordingTUI{}).Down(ctx, DownCommand{}))

		assert.Equal(t, []string{"web", "api"}, compose.stopped)
	})
}

func TestDown(t *testing.T) {
	t.Run("stops what up starts, and disabled projects still running", func(t *testing.T) {
		compose := &recordingCompose{running: map[string]bool{"web": true}}
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: newTestWorkspace()})

		require.NoError(t, NewManager(compose, &recordingTUI{}).Down(ctx, DownCommand{}))

		assert.Equal(t, []string{"web", "api", "local-env"}, compose.stopped)
	})

	t.Run("stops a disabled project when asked for it", func(t *testing.T) {
		compose := &recordingCompose{}
		ws := newTestWorkspace()
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[3]})

		require.NoError(t, NewManager(compose, &recordingTUI{}).Down(ctx, DownCommand{}))

		assert.Equal(t, []string{"admin"}, compose.stopped)
	})
}
