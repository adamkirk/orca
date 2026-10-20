package workspace

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHook(t *testing.T) {
	for _, name := range []string{"init.post", "clone.post", "up.pre", "up.post", "down.pre", "down.post"} {
		t.Run(name, func(t *testing.T) {
			hook, err := ParseHook(name)

			require.NoError(t, err)
			assert.Equal(t, name, hook.String())
		})
	}

	for _, name := range []string{"init.pre", "clone.pre", "exec.post", "up", ""} {
		t.Run("unknown "+name, func(t *testing.T) {
			_, err := ParseHook(name)

			assert.ErrorIs(t, err, ErrUnknownHook)
		})
	}
}

func TestLoadHooks(t *testing.T) {
	const wsPath = "/acme/local-env/orca.workspace.yaml"

	newRepo := func(t *testing.T, wsConfig string, projectConfig string, projectType string) *Repository {
		fs := afero.NewMemMapFs()

		require.NoError(t, afero.WriteFile(fs, wsPath, []byte(`
name: acme
projects:
  - name: api
    type: `+projectType+`
    repository:
      ssh: git@github.com:acme/api.git
`+wsConfig), 0644))
		require.NoError(t, afero.WriteFile(fs, "/acme/api/"+DefaultProjectFileName, []byte(projectConfig), 0644))

		return NewRepository(fs, stubUserConfig{
			workspaces: map[string]WorkspaceMeta{"acme": {Name: "acme", Path: wsPath}},
			projects: map[string]ProjectMeta{
				"api": {Name: "api", Path: "/acme/api", WorkspaceName: "acme", Type: ProjectType(projectType)},
			},
		})
	}

	const wsExtensions = `
extensions:
  - name: install-deps
    command: npm ci
  - name: seed
    command: ./scripts/seed.sh
`

	t.Run("loads the workspace's and projects' hooks, with their scopes", func(t *testing.T) {
		ws, err := newRepo(t, wsExtensions+`
hooks:
  init:
    post:
      - extension: seed
  up:
    pre:
      - extension: install-deps
        scope: workspace
`, `
extensions:
  - name: env
    type: syncFile
    source: .env.example
    target: .env
hooks:
  clone:
    post:
      - extension: env
      - extension: install-deps
        scope: workspace
  down:
    post:
      - extension: env
        scope: project
`, "service").Load("acme")

		require.NoError(t, err)
		assert.Equal(t, map[Hook][]HookEntry{
			HookInitPost: {{Extension: "seed", Scope: HookScopeWorkspace}},
			HookUpPre:    {{Extension: "install-deps", Scope: HookScopeWorkspace}},
		}, ws.Hooks)
		assert.Equal(t, map[Hook][]HookEntry{
			HookClonePost: {
				{Extension: "env", Scope: HookScopeProject},
				{Extension: "install-deps", Scope: HookScopeWorkspace},
			},
			HookDownPost: {{Extension: "env", Scope: HookScopeProject}},
		}, ws.Projects[0].Config.Hooks)
		assert.Equal(t, []Hook{HookInitPost, HookUpPre}, ws.DefinedHooks())
		assert.Equal(t, []Hook{HookClonePost, HookDownPost}, ws.Projects[0].DefinedHooks())
	})

	t.Run("lists every problem with the workspace's hooks", func(t *testing.T) {
		_, err := newRepo(t, wsExtensions+`
hooks:
  init:
    pre:
      - extension: seed
    post:
      - extension: seed
        scope: project
      - extension: missing
      - name: inline
        command: ./scripts/inline.sh
      - seed
  clone:
    post:
      - extension: seed
`, "", "service").Load("acme")

		assert.Equal(t, InvalidWorkspaceConfigError{
			Path: wsPath,
			Problems: []string{
				"hook 'clone.post' isn't supported in the workspace config (it supports: init.post, up.pre, up.post, down.pre, down.post)",
				"hook 'init.post', entry 1: 'scope' can't be 'project' in the workspace config, as its hooks aren't tied to a project",
				"hook 'init.post', entry 2: extension 'missing' isn't defined in the workspace config",
				"hook 'init.post', entry 3: 'name' can't be used in a hook (it takes: extension, scope); define the extension in 'extensions', and reference it by name",
				"hook 'init.post', entry 3: 'command' can't be used in a hook (it takes: extension, scope); define the extension in 'extensions', and reference it by name",
				"hook 'init.post', entry 3: 'extension' is required",
				"hook 'init.post', entry 4: must be a mapping, e.g. '- extension: env'",
				"hook 'init.pre' isn't supported in the workspace config (it supports: init.post, up.pre, up.post, down.pre, down.post)",
			},
		}, err)
	})

	t.Run("lists every problem with a project's hooks", func(t *testing.T) {
		_, err := newRepo(t, wsExtensions, `
extensions:
  - name: env
    type: syncFile
    source: .env.example
    target: .env
hooks:
  clone:
    pre:
      - extension: env
    post:
      - extension: seed
      - extension: env
        scope: everywhere
      - extension: nope
        scope: workspace
`, "service").Load("acme")

		assert.Equal(t, InvalidProjectConfigError{
			Path: "/acme/api/" + DefaultProjectFileName,
			Problems: []string{
				"hook 'clone.post', entry 1: extension 'seed' isn't defined in the project config",
				"hook 'clone.post', entry 2: 'scope' must be 'project' or 'workspace', not 'everywhere'",
				"hook 'clone.post', entry 3: extension 'nope' isn't defined in the workspace config",
				"hook 'clone.pre' isn't supported in a project config (it supports: clone.post, up.pre, up.post, down.pre, down.post)",
			},
		}, err)
	})

	t.Run("a library only has clone hooks", func(t *testing.T) {
		_, err := newRepo(t, wsExtensions, `
hooks:
  clone:
    post:
      - extension: install-deps
        scope: workspace
  up:
    pre:
      - extension: install-deps
        scope: workspace
`, "library").Load("acme")

		assert.Equal(t, InvalidProjectConfigError{
			Path: "/acme/api/" + DefaultProjectFileName,
			Problems: []string{
				"hook 'up.pre' isn't supported in a library project, as it's never started (it supports: clone.post)",
			},
		}, err)
	})
}
