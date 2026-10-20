package workspace

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestParseUnconfiguredWorkspace(t *testing.T) {
	repo := NewRepository(afero.NewMemMapFs(), nil)

	t.Run("valid config", func(t *testing.T) {
		contents := []byte(`
name: acme
projects:
  - name: root
    repository:
      self: true
  - name: svc-a
    repository:
      ssh: git@github.com:acme/svc-a.git
      branch: orca-adoption
`)

		ws, err := repo.ParseUnconfiguredWorkspace(contents, "orca.workspace.yaml")

		assert.NoError(t, err)
		assert.Equal(t, &UnconfiguredWorkspace{
			Name:            "acme",
			ConfigPath:      "orca.workspace.yaml",
			PreferencesPath: "orca.preferences.yaml",
			Projects: []UnconfiguredProject{
				{Name: "root", RepositoryConfig: ProjectRepositoryConfig{Self: true}},
				{Name: "svc-a", RepositoryConfig: ProjectRepositoryConfig{SSH: "git@github.com:acme/svc-a.git", Branch: "orca-adoption"}},
			},
		}, ws)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		_, err := repo.ParseUnconfiguredWorkspace([]byte("name: [acme"), "orca.workspace.yaml")

		assert.ErrorIs(t, err, ErrConfigIsNotValidYAML)
	})

	t.Run("wrong shape", func(t *testing.T) {
		_, err := repo.ParseUnconfiguredWorkspace([]byte("projects: acme"), "orca.workspace.yaml")

		assert.ErrorIs(t, err, ErrConfigIsNotValidYAML)
	})

	t.Run("missing required fields", func(t *testing.T) {
		contents := []byte(`
projects:
  - repository:
      ssh: git@github.com:acme/svc-a.git
  - name: svc-b
  - name: svc-c
    repository:
      self: true
      ssh: git@github.com:acme/svc-c.git
  - name: svc-d
    repository:
      self: true
      branch: orca-adoption
`)

		_, err := repo.ParseUnconfiguredWorkspace(contents, "orca.workspace.yaml")

		assert.Equal(t, InvalidWorkspaceConfigError{
			Path: "orca.workspace.yaml",
			Problems: []string{
				"'name' is required",
				"project at index 0: 'name' is required",
				"project 'svc-b': exactly one of 'repository.ssh' or 'repository.self' is required",
				"project 'svc-c': exactly one of 'repository.ssh' or 'repository.self' is required",
				"project 'svc-d': 'repository.branch' can't be combined with 'repository.self'",
			},
		}, err)
	})
}

type stubUserConfig struct {
	workspaces map[string]WorkspaceMeta
	projects   map[string]ProjectMeta
}

func (s stubUserConfig) GetWorkspaceMeta(name string) (WorkspaceMeta, error) {
	meta, ok := s.workspaces[name]

	if !ok {
		return WorkspaceMeta{}, fmt.Errorf("%w: %s", ErrUnknownWorkspace, name)
	}

	return meta, nil
}

func (s stubUserConfig) GetProjectMeta(wsName string, projectName string) (ProjectMeta, error) {
	meta, ok := s.projects[projectName]

	if !ok {
		return ProjectMeta{}, fmt.Errorf("%w: %s", ErrUnknownProject, projectName)
	}

	return meta, nil
}

func TestLoadValidatesConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/acme/orca.workspace.yaml"

	assert.NoError(t, afero.WriteFile(fs, path, []byte("projects:\n  - name: api\n"), 0644))

	repo := NewRepository(fs, stubUserConfig{
		workspaces: map[string]WorkspaceMeta{
			"acme": {Name: "acme", Path: path},
		},
	})

	_, err := repo.Load("acme")

	assert.Equal(t, InvalidWorkspaceConfigError{
		Path: path,
		Problems: []string{
			"'name' is required",
			"project 'api': exactly one of 'repository.ssh' or 'repository.self' is required",
		},
	}, err)
}

func TestLoadRepositoryConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/acme/root/orca.workspace.yaml"

	assert.NoError(t, afero.WriteFile(fs, path, []byte(`
name: acme
projects:
  - name: root
    repository:
      self: true
  - name: api
    repository:
      ssh: git@github.com:acme/api.git
      branch: orca-adoption
  - name: web
    repository:
      ssh: git@github.com:acme/web.git
      branch: orca-adoption
`), 0644))
	assert.NoError(t, afero.WriteFile(fs, "/acme/api/"+DefaultProjectFileName, []byte("hosts: []\n"), 0644))

	repo := NewRepository(fs, stubUserConfig{
		workspaces: map[string]WorkspaceMeta{
			"acme": {Name: "acme", Path: path},
		},
		projects: map[string]ProjectMeta{
			"api": {Name: "api", Path: "/acme/api", WorkspaceName: "acme"},
		},
	})

	ws, err := repo.Load("acme")

	assert.NoError(t, err)

	t.Run("registered project", func(t *testing.T) {
		assert.Equal(t, ProjectRepositoryConfig{SSH: "git@github.com:acme/api.git", Branch: "orca-adoption"}, ws.Projects[1].RepositoryConfig)
	})

	t.Run("unregistered project", func(t *testing.T) {
		assert.Equal(t, ProjectRepositoryConfig{SSH: "git@github.com:acme/web.git", Branch: "orca-adoption"}, ws.Projects[2].RepositoryConfig)
	})

	t.Run("self project", func(t *testing.T) {
		assert.Equal(t, ProjectRepositoryConfig{Self: true}, ws.Projects[0].RepositoryConfig)
	})
}

func TestLoadExtensions(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/acme/api/orca.workspace.yaml"

	assert.NoError(t, afero.WriteFile(fs, path, []byte(`
name: acme
projects:
  - name: api
    repository:
      self: true
`), 0644))
	assert.NoError(t, afero.WriteFile(fs, "/acme/api/"+DefaultProjectFileName, []byte(`
extensions:
  - name: artisan
    service: app
    command: php artisan
    description: Run an artisan command
  - name: test
    service: app
    command: vendor/bin/phpunit
`), 0644))

	repo := NewRepository(fs, stubUserConfig{
		workspaces: map[string]WorkspaceMeta{
			"acme": {Name: "acme", Path: path},
		},
		projects: map[string]ProjectMeta{
			"api": {Name: "api", Path: "/acme/api", WorkspaceName: "acme"},
		},
	})

	ws, err := repo.Load("acme")

	assert.NoError(t, err)
	assert.Equal(t, []Extension{
		{Name: "artisan", Service: "app", Command: "php artisan", Description: "Run an artisan command"},
		{Name: "test", Service: "app", Command: "vendor/bin/phpunit"},
	}, ws.Projects[0].Config.Extensions)
}

func TestLoadPreferences_AppliesToProjects(t *testing.T) {
	newRepo := func(t *testing.T, wsConfig string, prefsPath string, prefs string) *Repository {
		fs := afero.NewMemMapFs()
		path := "/acme/root/orca.workspace.yaml"

		assert.NoError(t, afero.WriteFile(fs, path, []byte(wsConfig), 0644))

		if prefs != "" {
			assert.NoError(t, afero.WriteFile(fs, prefsPath, []byte(prefs), 0644))
		}

		return NewRepository(fs, stubUserConfig{
			workspaces: map[string]WorkspaceMeta{"acme": {Name: "acme", Path: path}},
		})
	}

	wsConfig := `
name: acme
projects:
  - name: api
    repository:
      ssh: git@github.com:acme/api.git
  - name: web
    repository:
      ssh: git@github.com:acme/web.git
  - name: admin
    repository:
      ssh: git@github.com:acme/admin.git
`

	t.Run("disables projects, ignoring unknown ones", func(t *testing.T) {
		repo := newRepo(t, wsConfig, "/acme/root/orca.preferences.yaml", `
projects:
  api:
    enabled: true
  web:
    enabled: false
  removed:
    enabled: false
  gone:
`)

		ws, err := repo.Load("acme")

		assert.NoError(t, err)
		assert.Equal(t, "/acme/root/orca.preferences.yaml", ws.PreferencesPath)
		assert.False(t, ws.Projects[0].Disabled)
		assert.True(t, ws.Projects[1].Disabled)
		assert.False(t, ws.Projects[2].Disabled, "projects that aren't listed are enabled")
		assert.Equal(t, []string{"gone", "removed"}, ws.UnknownPreferenceProjects)
	})

	t.Run("a missing file enables every project", func(t *testing.T) {
		ws, err := newRepo(t, wsConfig, "", "").Load("acme")

		assert.NoError(t, err)
		assert.Empty(t, ws.UnknownPreferenceProjects)

		for _, p := range ws.Projects {
			assert.False(t, p.Disabled)
		}
	})

	t.Run("the workspace config can move the file", func(t *testing.T) {
		repo := newRepo(t, wsConfig+"preferences:\n  path: config/prefs.yaml\n", "/acme/root/config/prefs.yaml", "projects:\n  admin:\n    enabled: false\n")

		ws, err := repo.Load("acme")

		assert.NoError(t, err)
		assert.Equal(t, "/acme/root/config/prefs.yaml", ws.PreferencesPath)
		assert.True(t, ws.Projects[2].Disabled)
	})

	t.Run("the workspace config sets the default, and preferences override it", func(t *testing.T) {
		defaults := `
name: acme
projects:
  - name: api
    repository:
      ssh: git@github.com:acme/api.git
    enabled: false
  - name: web
    repository:
      ssh: git@github.com:acme/web.git
    enabled: false
  - name: admin
    repository:
      ssh: git@github.com:acme/admin.git
    enabled: true
  - name: docs
    repository:
      ssh: git@github.com:acme/docs.git
`
		repo := newRepo(t, defaults, "/acme/root/orca.preferences.yaml", `
projects:
  api:
    enabled: true
  web:
  admin:
    enabled: false
`)

		ws, err := repo.Load("acme")

		assert.NoError(t, err)
		assert.False(t, ws.Projects[0].Disabled, "enabled by the preferences")
		assert.True(t, ws.Projects[1].Disabled, "listed without 'enabled', so the default applies")
		assert.True(t, ws.Projects[2].Disabled, "disabled by the preferences")
		assert.False(t, ws.Projects[3].Disabled, "enabled when neither says")
	})

	t.Run("without preferences, the workspace config's default applies", func(t *testing.T) {
		ws, err := newRepo(t, wsConfig+"    enabled: false\n", "", "").Load("acme")

		assert.NoError(t, err)
		assert.True(t, ws.Projects[2].Disabled)
	})

	t.Run("an invalid file fails with its problems", func(t *testing.T) {
		repo := newRepo(t, wsConfig, "/acme/root/orca.preferences.yaml", "projects:\n  api:\n    enabled: nope\n")

		_, err := repo.Load("acme")

		assert.Equal(t, InvalidPreferencesError{
			Path:     "/acme/root/orca.preferences.yaml",
			Problems: []string{"project 'api': 'enabled' must be true or false"},
		}, err)
	})
}
