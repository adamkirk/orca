package repository

import (
	"testing"

	"github.com/adamkirk/orca/internal/common"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestParseUnconfiguredWorkspace(t *testing.T) {
	repo := NewWorkspaceRepository(afero.NewMemMapFs(), nil)

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
		assert.Equal(t, &common.UnconfiguredWorkspace{
			Name:       "acme",
			ConfigPath: "orca.workspace.yaml",
			Projects: []common.UnconfiguredProject{
				{Name: "root", RepositoryConfig: common.ProjectRepositoryConfig{Self: true}},
				{Name: "svc-a", RepositoryConfig: common.ProjectRepositoryConfig{SSH: "git@github.com:acme/svc-a.git"}},
			},
		}, ws)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		_, err := repo.ParseUnconfiguredWorkspace([]byte("name: [acme"), "orca.workspace.yaml")

		assert.IsType(t, common.ErrConfigIsNotValidYAML{}, err)
	})

	t.Run("wrong shape", func(t *testing.T) {
		_, err := repo.ParseUnconfiguredWorkspace([]byte("projects: acme"), "orca.workspace.yaml")

		assert.IsType(t, common.ErrConfigIsNotValidYAML{}, err)
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

		assert.Equal(t, common.ErrInvalidWorkspaceConfig{
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
	workspaces map[string]common.WorkspaceMeta
}

func (s stubUserConfig) GetWorkspaceMeta(name string) (common.WorkspaceMeta, error) {
	meta, ok := s.workspaces[name]

	if !ok {
		return common.WorkspaceMeta{}, common.ErrUnknownWorkspace{Name: name}
	}

	return meta, nil
}

func (s stubUserConfig) GetProjectMeta(wsName string, projectName string) (common.ProjectMeta, error) {
	return common.ProjectMeta{}, common.ErrUnknownProject{Name: projectName}
}

func TestLoadValidatesConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/acme/orca.workspace.yaml"

	assert.NoError(t, afero.WriteFile(fs, path, []byte("projects:\n  - name: api\n"), 0644))

	repo := NewWorkspaceRepository(fs, stubUserConfig{
		workspaces: map[string]common.WorkspaceMeta{
			"acme": {Name: "acme", Path: path},
		},
	})

	_, err := repo.Load("acme")

	assert.Equal(t, common.ErrInvalidWorkspaceConfig{
		Path: path,
		Problems: []string{
			"'name' is required",
			"project 'api': exactly one of 'repository.ssh' or 'repository.self' is required",
		},
	}, err)
}
