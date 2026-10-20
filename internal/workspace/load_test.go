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
			Name:       "acme",
			ConfigPath: "orca.workspace.yaml",
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
