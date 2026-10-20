package workspace

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnable(t *testing.T) {
	prefsPath := "/acme/root/orca.preferences.yaml"

	newCtx := func() context.Context {
		return WithExecutionContext(context.Background(), ExecutionContext{Workspace: &Workspace{
			Name:            "acme",
			PreferencesPath: prefsPath,
			Projects:        []Project{{Name: "api"}, {Name: "web", Disabled: true}, {Name: "admin", Disabled: true}},
		}})
	}

	t.Run("enables projects", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, prefsPath, []byte("projects:\n  web:\n    enabled: false\n  admin:\n    enabled: false\n"), 0644))
		tui := &recordingTUI{}

		err := NewManager(fs, tui, nil, nil, nil, nil, nil, nil).Enable(newCtx(), EnableCommand{Projects: []string{"web", "admin", "web"}})

		require.NoError(t, err)
		assert.Equal(t, []string{"Enabled web and admin."}, tui.lines)

		prefs, err := loadPreferences(fs, prefsPath)
		require.NoError(t, err)
		assert.True(t, *prefs.Projects["web"].Enabled)
		assert.True(t, *prefs.Projects["admin"].Enabled)
	})

	t.Run("unknown projects change nothing", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		err := NewManager(fs, &recordingTUI{}, nil, nil, nil, nil, nil, nil).Enable(newCtx(), EnableCommand{Projects: []string{"web", "typo", "other"}})

		assert.ErrorIs(t, err, ErrUnknownProject)
		assert.EqualError(t, err, "Workspace 'acme' has no project typo or other!: unknown project: typo, other")

		exists, _ := afero.Exists(fs, prefsPath)
		assert.False(t, exists)
	})

	t.Run("no projects", func(t *testing.T) {
		err := NewManager(afero.NewMemMapFs(), &recordingTUI{}, nil, nil, nil, nil, nil, nil).Enable(newCtx(), EnableCommand{})

		assert.ErrorIs(t, err, ErrInvalidInput)
	})
}

func TestJoinNames(t *testing.T) {
	assert.Equal(t, "", joinNames(nil, "and"))
	assert.Equal(t, "web", joinNames([]string{"web"}, "and"))
	assert.Equal(t, "web and admin", joinNames([]string{"web", "admin"}, "and"))
	assert.Equal(t, "web, admin or api", joinNames([]string{"web", "admin", "api"}, "or"))
}
