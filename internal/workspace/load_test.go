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
`)

		ws, err := repo.ParseUnconfiguredWorkspace(contents, "orca.workspace.yaml")

		assert.NoError(t, err)
		assert.Equal(t, &UnconfiguredWorkspace{
			Name:       "acme",
			ConfigPath: "orca.workspace.yaml",
			Projects: []UnconfiguredProject{
				{Name: "root", RepositoryConfig: ProjectRepositoryConfig{Self: true}},
				{Name: "svc-a", RepositoryConfig: ProjectRepositoryConfig{SSH: "git@github.com:acme/svc-a.git"}},
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
`)

		_, err := repo.ParseUnconfiguredWorkspace(contents, "orca.workspace.yaml")

		assert.Equal(t, InvalidWorkspaceConfigError{
			Path: "orca.workspace.yaml",
			Problems: []string{
				"'name' is required",
				"project at index 0: 'name' is required",
				"project 'svc-b': exactly one of 'repository.ssh' or 'repository.self' is required",
				"project 'svc-c': exactly one of 'repository.ssh' or 'repository.self' is required",
			},
		}, err)
	})
}

type stubUserConfig struct {
	workspaces map[string]WorkspaceMeta
}

func (s stubUserConfig) GetWorkspaceMeta(name string) (WorkspaceMeta, error) {
	meta, ok := s.workspaces[name]

	if !ok {
		return WorkspaceMeta{}, fmt.Errorf("%w: %s", ErrUnknownWorkspace, name)
	}

	return meta, nil
}

func (s stubUserConfig) GetProjectMeta(wsName string, projectName string) (ProjectMeta, error) {
	return ProjectMeta{}, fmt.Errorf("%w: %s", ErrUnknownProject, projectName)
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
