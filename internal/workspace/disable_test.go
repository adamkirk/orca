package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisable(t *testing.T) {
	prefsPath := "/acme/root/orca.preferences.yaml"

	newWorkspace := func() *Workspace {
		return &Workspace{
			Name:            "acme",
			PreferencesPath: prefsPath,
			Projects: []Project{
				{Name: "local-env"},
				{Name: "api", Requires: []string{"local-env"}},
				{Name: "web", Requires: []string{"api"}},
				{Name: "admin"},
				{Name: "sdk", Type: ProjectTypeLibrary},
			},
		}
	}

	tests := []struct {
		name        string
		cmd         DisableCommand
		running     map[string]bool
		stackErr    error
		wantStopped []string
		wantLines   []string
	}{
		{
			name:      "a project that isn't running is only disabled",
			cmd:       DisableCommand{Projects: []string{"admin"}},
			running:   map[string]bool{},
			wantLines: []string{"Disabled admin."},
		},
		{
			name:        "running projects are stopped, before what they require",
			cmd:         DisableCommand{Projects: []string{"api", "web"}},
			running:     map[string]bool{"local-env": true, "api": true, "web": true},
			wantStopped: []string{"web", "api"},
			wantLines:   []string{"Disabled api and web."},
		},
		{
			name:    "orphan leaves them running",
			cmd:     DisableCommand{Projects: []string{"web"}, Orphan: true},
			running: map[string]bool{"web": true},
			wantLines: []string{
				"Disabled web.",
				"web is still running, stop it with 'orca down -p web'.",
			},
		},
		{
			name:    "a project a running, enabled project requires is left running",
			cmd:     DisableCommand{Projects: []string{"local-env"}},
			running: map[string]bool{"local-env": true, "api": true},
			wantLines: []string{
				"Disabled local-env.",
				"Warning: api requires local-env, so it's been left running.",
			},
		},
		{
			name:        "a requirement of a stopped project is stopped",
			cmd:         DisableCommand{Projects: []string{"local-env", "api"}},
			running:     map[string]bool{"local-env": true, "api": true},
			wantStopped: []string{"api", "local-env"},
			wantLines:   []string{"Disabled local-env and api."},
		},
		{
			name:      "a library has nothing to stop, so docker isn't asked",
			cmd:       DisableCommand{Projects: []string{"sdk"}},
			stackErr:  errors.New("docker: command not found"),
			wantLines: []string{"Disabled sdk."},
		},
		{
			name:     "without docker, they're only disabled",
			cmd:      DisableCommand{Projects: []string{"admin"}},
			stackErr: errors.New("docker: command not found"),
			wantLines: []string{
				"Disabled admin.",
				"Warning: couldn't check whether admin is running, so it's been left as it is.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			tui := &recordingTUI{}
			stack := &stubStack{running: tt.running, err: tt.stackErr}
			ctx := WithExecutionContext(context.Background(), ExecutionContext{Workspace: newWorkspace()})

			err := NewManager(fs, tui, nil, nil, nil, nil, stack).Disable(ctx, tt.cmd)

			require.NoError(t, err)
			assert.Equal(t, tt.wantStopped, stack.stopped)
			assert.Equal(t, tt.wantLines, tui.lines)

			prefs, err := loadPreferences(fs, prefsPath)
			require.NoError(t, err)

			for _, name := range tt.cmd.Projects {
				assert.False(t, *prefs.Projects[name].Enabled)
			}
		})
	}

	t.Run("unknown projects change nothing", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		stack := &stubStack{running: map[string]bool{"web": true}}
		ctx := WithExecutionContext(context.Background(), ExecutionContext{Workspace: newWorkspace()})

		err := NewManager(fs, &recordingTUI{}, nil, nil, nil, nil, stack).Disable(ctx, DisableCommand{Projects: []string{"web", "typo"}})

		assert.ErrorIs(t, err, ErrUnknownProject)
		assert.Empty(t, stack.stopped)

		exists, _ := afero.Exists(fs, prefsPath)
		assert.False(t, exists)
	})
}
