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

	// local-env is the workspace's own repository, and an ordinary project
	newRepo := func(t *testing.T, wsHooks string, rootConfig string, apiConfig string, apiType string) *Repository {
		fs := afero.NewMemMapFs()

		require.NoError(t, afero.WriteFile(fs, wsPath, []byte(`
name: acme
projects:
  - name: local-env
    repository:
      self: true
  - name: api
    type: `+apiType+`
    repository:
      ssh: git@github.com:acme/api.git
  - name: web
    repository:
      ssh: git@github.com:acme/web.git
`+wsHooks), 0644))
		require.NoError(t, afero.WriteFile(fs, "/acme/local-env/"+DefaultProjectFileName, []byte(rootConfig), 0644))
		require.NoError(t, afero.WriteFile(fs, "/acme/api/"+DefaultProjectFileName, []byte(apiConfig), 0644))

		return NewRepository(fs, stubUserConfig{
			workspaces: map[string]WorkspaceMeta{"acme": {Name: "acme", Path: wsPath}},
			projects: map[string]ProjectMeta{
				"local-env": {Name: "local-env", Path: "/acme/local-env", WorkspaceName: "acme"},
				"api":       {Name: "api", Path: "/acme/api", WorkspaceName: "acme", Type: ProjectType(apiType)},
			},
		})
	}

	const rootConfig = `
extensions:
  - name: seed
    command: ./scripts/seed.sh
  - name: check-tools
    command: ./scripts/check-tools.sh
hooks:
  up:
    pre:
      - extension: check-tools
`

	const apiConfig = `
extensions:
  - name: env
    type: syncFile
    source: .env.example
    target: .env
hooks:
  clone:
    post:
      - extension: env
  down:
    post:
      - extension: env
`

	t.Run("loads the workspace's init.post, and every project's own hooks", func(t *testing.T) {
		ws, err := newRepo(t, `
hooks:
  init:
    post:
      - extension: seed
        project: local-env
      - extension: env
        project: api
`, rootConfig, apiConfig, "service").Load("acme")

		require.NoError(t, err)
		assert.Equal(t, []InitHookEntry{{Project: "local-env", Extension: "seed"}, {Project: "api", Extension: "env"}}, ws.InitHooks)

		root, _ := ws.GetProject("local-env")
		assert.Equal(t, map[Hook][]HookEntry{HookUpPre: {{Extension: "check-tools"}}}, root.Config.Hooks)

		api, _ := ws.GetProject("api")
		assert.Equal(t, map[Hook][]HookEntry{
			HookClonePost: {{Extension: "env"}},
			HookDownPost:  {{Extension: "env"}},
		}, api.Config.Hooks)
		assert.Equal(t, []Hook{HookClonePost, HookDownPost}, api.DefinedHooks())
	})

	t.Run("the workspace config can only have init.post, naming a project", func(t *testing.T) {
		_, err := newRepo(t, `
hooks:
  init:
    post:
      - extension: seed
      - extension: seed
        project: nope
      - project: local-env
      - extension: seed
        project: local-env
        scope: workspace
  up:
    pre:
      - extension: check-tools
        project: local-env
`, rootConfig, apiConfig, "service").Load("acme")

		assert.Equal(t, InvalidWorkspaceConfigError{
			Path: wsPath,
			Problems: []string{
				"hook 'init.post', entry 1: 'project' is required, naming the project whose extension it runs, as the workspace config has no extensions",
				"hook 'init.post', entry 2: project 'nope' isn't in the workspace",
				"hook 'init.post', entry 3: 'extension' is required",
				"hook 'init.post', entry 4: 'scope' can't be used in a hook (it takes: extension, project); define the extension in a project's 'extensions', and reference it by name",
				"hook 'up.pre' isn't supported in the workspace config (it supports: init.post); put the others in a project's config",
			},
		}, err)
	})

	t.Run("init.post's extensions must be defined by their projects", func(t *testing.T) {
		_, err := newRepo(t, `
hooks:
  init:
    post:
      - extension: seed
        project: api
      - extension: deploy
        project: web
`, rootConfig, apiConfig, "service").Load("acme")

		// web isn't cloned, so it can't be checked yet
		assert.Equal(t, InvalidWorkspaceConfigError{
			Path:     wsPath,
			Problems: []string{"hook 'init.post', entry 1: project 'api' doesn't define extension 'seed'"},
		}, err)
	})

	t.Run("a project's hooks can only run its own extensions", func(t *testing.T) {
		_, err := newRepo(t, "", rootConfig, `
extensions:
  - name: env
    type: syncFile
    source: .env.example
    target: .env
hooks:
  init:
    post:
      - extension: env
  clone:
    pre:
      - extension: env
    post:
      - extension: seed
      - extension: env
        scope: workspace
      - name: inline
        command: ./scripts/inline.sh
      - env
`, "service").Load("acme")

		assert.Equal(t, InvalidProjectConfigError{
			Path: "/acme/api/" + DefaultProjectFileName,
			Problems: []string{
				"hook 'clone.post', entry 1: extension 'seed' isn't defined in this config",
				"hook 'clone.post', entry 2: 'scope' can't be used in a hook (it takes: extension); define the extension in a project's 'extensions', and reference it by name",
				"hook 'clone.post', entry 3: 'name' can't be used in a hook (it takes: extension); define the extension in a project's 'extensions', and reference it by name",
				"hook 'clone.post', entry 3: 'command' can't be used in a hook (it takes: extension); define the extension in a project's 'extensions', and reference it by name",
				"hook 'clone.post', entry 3: 'extension' is required",
				"hook 'clone.post', entry 4: must be a mapping, e.g. '- extension: env'",
				"hook 'clone.pre' isn't supported in a project config (it supports: clone.post, up.pre, up.post, down.pre, down.post)",
				"hook 'init.post' isn't supported in a project config (it supports: clone.post, up.pre, up.post, down.pre, down.post)",
			},
		}, err)
	})

	t.Run("a library only has clone hooks", func(t *testing.T) {
		_, err := newRepo(t, "", rootConfig, `
extensions:
  - name: build
    command: make build
hooks:
  clone:
    post:
      - extension: build
  up:
    pre:
      - extension: build
`, "library").Load("acme")

		assert.Equal(t, InvalidProjectConfigError{
			Path: "/acme/api/" + DefaultProjectFileName,
			Problems: []string{
				"hook 'up.pre' isn't supported in a library project, as it's never started (it supports: clone.post)",
			},
		}, err)
	})
}
