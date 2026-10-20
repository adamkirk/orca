package workspace

import (
	"testing"

	"github.com/adamkirk/orca/internal/workspace/internal/model"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreferencesPath(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		want       string
	}{
		{name: "next to the workspace config by default", want: "/acme/root/orca.preferences.yaml"},
		{name: "relative to the workspace config", configured: "config/prefs.yaml", want: "/acme/root/config/prefs.yaml"},
		{name: "outside the workspace repo", configured: "../prefs.yaml", want: "/acme/prefs.yaml"},
		{name: "absolute", configured: "/home/me/prefs.yaml", want: "/home/me/prefs.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, preferencesPath("/acme/root/orca.workspace.yaml", tt.configured))
		})
	}
}

func TestLoadPreferences(t *testing.T) {
	no := false
	yes := true
	path := "/acme/orca.preferences.yaml"

	tests := []struct {
		name     string
		contents *string
		want     *model.Preferences
		wantErr  error

		wantProblems []string
	}{
		{
			name: "a missing file enables everything",
			want: &model.Preferences{},
		},
		{
			name:     "an empty file enables everything",
			contents: new(""),
			want:     &model.Preferences{},
		},
		{
			name:     "projects without settings",
			contents: new("projects:\n  api:\n"),
			want:     &model.Preferences{Projects: map[string]model.ProjectPreferences{"api": {}}},
		},
		{
			name:     "enabled and disabled projects",
			contents: new("projects:\n  api:\n    enabled: true\n  web:\n    enabled: false\n"),
			want: &model.Preferences{Projects: map[string]model.ProjectPreferences{
				"api": {Enabled: &yes},
				"web": {Enabled: &no},
			}},
		},
		{
			name:     "invalid yaml",
			contents: new("projects: [api"),
			wantErr:  ErrConfigIsNotValidYAML,
		},
		{
			name:     "every problem is listed",
			contents: new("colour: blue\nprojects:\n  api:\n    enabled: maybe\n    port: 80\n  web: false\n"),
			wantProblems: []string{
				"unknown property 'colour'",
				"project 'api': 'enabled' must be true or false",
				"project 'api': unknown property 'port'",
				"project 'web': must be a map, e.g. 'enabled: true'",
			},
		},
		{
			name:         "projects must be a map",
			contents:     new("projects:\n  - api\n"),
			wantProblems: []string{"'projects' must be a map of project names"},
		},
		{
			name:         "the file must be a map",
			contents:     new("- api\n"),
			wantProblems: []string{"the file must be a map, e.g. 'projects:'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			if tt.contents != nil {
				require.NoError(t, afero.WriteFile(fs, path, []byte(*tt.contents), 0644))
			}

			got, err := loadPreferences(fs, path)

			if tt.wantProblems != nil {
				assert.Equal(t, InvalidPreferencesError{Path: path, Problems: tt.wantProblems}, err)
				return
			}

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSetProjectsEnabled(t *testing.T) {
	path := "/acme/config/orca.preferences.yaml"

	t.Run("creates the file and its directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		require.NoError(t, setProjectsEnabled(fs, path, []string{"web", "admin"}, false))

		contents, err := afero.ReadFile(fs, path)
		require.NoError(t, err)
		assert.Equal(t, `# Your own preferences for this workspace, such as which projects are enabled.
# A project without 'enabled' set follows the workspace config, which enables it unless it says otherwise.

projects:
  web:
    enabled: false
  admin:
    enabled: false
`, string(contents))
	})

	t.Run("keeps the rest of the file, and its comments", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, path, []byte(`# My settings
projects:
  # The API
  api:
    enabled: true
  web:
`), 0644))

		require.NoError(t, setProjectsEnabled(fs, path, []string{"api", "web", "admin"}, false))

		contents, err := afero.ReadFile(fs, path)
		require.NoError(t, err)
		assert.Equal(t, `# My settings
projects:
  # The API
  api:
    enabled: false
  web:
    enabled: false
  admin:
    enabled: false
`, string(contents))
	})

	t.Run("an invalid file isn't changed", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, path, []byte("projects: api\n"), 0644))

		err := setProjectsEnabled(fs, path, []string{"api"}, false)

		assert.ErrorAs(t, err, new(InvalidPreferencesError))

		contents, err := afero.ReadFile(fs, path)
		require.NoError(t, err)
		assert.Equal(t, "projects: api\n", string(contents))
	})
}

func TestWriteDefaultPreferences(t *testing.T) {
	path := "/acme/orca.preferences.yaml"

	t.Run("lists every project, leaving them to the workspace config", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		created, err := writeDefaultPreferences(fs, path, []string{"api", "web"})

		require.NoError(t, err)
		assert.True(t, created)

		contents, err := afero.ReadFile(fs, path)
		require.NoError(t, err)
		assert.Equal(t, `# Your own preferences for this workspace, such as which projects are enabled.
# A project without 'enabled' set follows the workspace config, which enables it unless it says otherwise.

projects:
  api:
  web:
`, string(contents))

		prefs, err := loadPreferences(fs, path)
		require.NoError(t, err)
		assert.Equal(t, map[string]model.ProjectPreferences{"api": {}, "web": {}}, prefs.Projects)
	})

	t.Run("never overwrites an existing file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, path, []byte("projects:\n  web:\n    enabled: false\n"), 0644))

		created, err := writeDefaultPreferences(fs, path, []string{"api", "web"})

		require.NoError(t, err)
		assert.False(t, created)

		contents, err := afero.ReadFile(fs, path)
		require.NoError(t, err)
		assert.Equal(t, "projects:\n  web:\n    enabled: false\n", string(contents))
	})
}
