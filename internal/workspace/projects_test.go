package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingTUI keeps what's shown, a table as its header and rows.
type recordingTUI struct {
	lines  []string
	header []string
	rows   [][]string
}

func (r *recordingTUI) Info(msg ...string)    { r.lines = append(r.lines, msg...) }
func (r *recordingTUI) Error(msg ...string)   { r.lines = append(r.lines, msg...) }
func (r *recordingTUI) Success(msg ...string) { r.lines = append(r.lines, msg...) }
func (r *recordingTUI) Table(header []string, rows [][]string) {
	r.header = header
	r.rows = rows
}

// stubStack knows which projects are running, and records those stopped.
type stubStack struct {
	running map[string]bool
	err     error
	stopped []string
}

func (s *stubStack) IsProjectRunning(_ *Workspace, p *Project) (bool, error) {
	return s.running[p.Name], s.err
}

func (s *stubStack) Down(_ *Workspace, p *Project) error {
	s.stopped = append(s.stopped, p.Name)
	s.running[p.Name] = false

	return nil
}

func TestProjects(t *testing.T) {
	newWorkspace := func() *Workspace {
		return &Workspace{
			Name:            "acme",
			PreferencesPath: "/acme/root/orca.preferences.yaml",
			Projects: []Project{
				{Name: "api", IsRegistered: true, ProjectDir: "/acme/api"},
				{Name: "web", IsRegistered: true, ProjectDir: "/acme/web", Disabled: true},
				{Name: "admin"},
			},
		}
	}

	newManager := func(t *testing.T, stack *stubStack) (*Manager, *recordingTUI) {
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/acme/api", 0755))
		require.NoError(t, fs.MkdirAll("/acme/web", 0755))
		tui := &recordingTUI{}

		return NewManager(fs, tui, nil, nil, nil, nil, stack), tui
	}

	t.Run("lists each project", func(t *testing.T) {
		m, tui := newManager(t, &stubStack{running: map[string]bool{"api": true}})
		ctx := WithExecutionContext(context.Background(), ExecutionContext{Workspace: newWorkspace()})

		require.NoError(t, m.Projects(ctx, ProjectsQuery{}))

		assert.Equal(t, []string{"Project", "Type", "Enabled", "Cloned", "Running"}, tui.header)
		assert.Equal(t, [][]string{
			{"api", "service", "yes", "yes", "yes"},
			{"web", "service", "no", "yes", "no"},
			{"admin", "service", "yes", "no", "no"},
		}, tui.rows)
		assert.Empty(t, tui.lines)
	})

	t.Run("running is unknown without docker", func(t *testing.T) {
		m, tui := newManager(t, &stubStack{err: errors.New("docker: command not found")})
		ctx := WithExecutionContext(context.Background(), ExecutionContext{Workspace: newWorkspace()})

		require.NoError(t, m.Projects(ctx, ProjectsQuery{}))

		for _, row := range tui.rows {
			assert.Equal(t, "?", row[4])
		}
	})

	t.Run("warns about unknown projects in the preferences", func(t *testing.T) {
		m, tui := newManager(t, &stubStack{})
		ws := newWorkspace()
		ws.UnknownPreferenceProjects = []string{"removed"}

		require.NoError(t, m.Projects(WithExecutionContext(context.Background(), ExecutionContext{Workspace: ws}), ProjectsQuery{}))

		assert.Equal(t, []string{
			"Warning: /acme/root/orca.preferences.yaml lists 'removed', which isn't in the workspace, so it's ignored.",
		}, tui.lines)
	})
}
