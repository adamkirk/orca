package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGithubRepo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected githubRepo
		wantErr  bool
	}{
		{
			name:  "shorthand",
			input: "acme/workspace",
			expected: githubRepo{
				Owner:    "acme",
				Name:     "workspace",
				CloneURL: "git@github.com:acme/workspace.git",
			},
		},
		{
			name:    "ssh url",
			input:   "git@github.com:acme/workspace.git",
			wantErr: true,
		},
		{
			name:    "missing owner",
			input:   "workspace",
			wantErr: true,
		},
		{
			name:    "empty repo",
			input:   "acme/",
			wantErr: true,
		},
		{
			name:    "https url",
			input:   "https://github.com/acme/workspace",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := parseGithubRepo(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

// stubGithub knows the workspace config on each branch of one repository.
type stubGithub struct {
	configs map[string]string
	refs    []string
}

func (g *stubGithub) GetFileContents(owner string, repo string, path string, ref string) ([]byte, bool, error) {
	g.refs = append(g.refs, ref)

	contents, ok := g.configs[ref]

	return []byte(contents), ok, nil
}

func TestInitialiseFromBranch(t *testing.T) {
	newManager := func() (*Manager, *stubGit, *stubGithub) {
		fs := afero.NewMemMapFs()
		git := &stubGit{
			branches: map[string][]string{
				"git@github.com:acme/api.git":       {"main", "orca-adoption"},
				"git@github.com:acme/web.git":       {"main"},
				"git@github.com:acme/local-env.git": {"main", "orca-adoption"},
			},
		}
		github := &stubGithub{
			configs: map[string]string{
				"orca-adoption": `
name: acme
projects:
  - name: api
    repository:
      ssh: git@github.com:acme/api.git
      branch: orca-adoption
  - name: web
    repository:
      ssh: git@github.com:acme/web.git
      branch: orca-adoption
`,
			},
		}

		return NewManager(fs, stubTUI{}, nil, git, github, NewRepository(fs, nil), nil, nil), git, github
	}

	t.Run("missing workspace branch", func(t *testing.T) {
		m, git, github := newManager()

		err := m.Initialise(context.Background(), InitialiseCommand{
			SourceRepo:        "acme/local-env",
			Branch:            "typo",
			WorkspaceFileName: DefaultWorkspaceFileName,
			Into:              "/acme",
		})

		assert.ErrorIs(t, err, ErrBranchNotFound)
		assert.ErrorContains(t, err, "acme/local-env: 'typo'")
		assert.Empty(t, github.refs)
		assert.Empty(t, git.cloned)
	})

	t.Run("missing project branch stops before the workspace repo is cloned", func(t *testing.T) {
		m, git, github := newManager()

		err := m.Initialise(context.Background(), InitialiseCommand{
			SourceRepo:        "acme/local-env",
			Branch:            "orca-adoption",
			WorkspaceFileName: DefaultWorkspaceFileName,
			Into:              "/acme",
		})

		assert.ErrorIs(t, err, ErrBranchNotFound)
		assert.ErrorContains(t, err, "web: 'orca-adoption' (git@github.com:acme/web.git)")
		assert.Equal(t, []string{"orca-adoption"}, github.refs)
		assert.Empty(t, git.cloned)
	})

	t.Run("missing project branch stops a local init before it's registered", func(t *testing.T) {
		m, git, _ := newManager()

		// There's no config manager, so registering the workspace would panic
		assert.NoError(t, afero.WriteFile(m.fs, "/acme/local-env/"+DefaultWorkspaceFileName, []byte(`
name: acme
projects:
  - name: web
    repository:
      ssh: git@github.com:acme/web.git
      branch: orca-adoption
`), 0644))

		err := m.Initialise(context.Background(), InitialiseCommand{
			SourceDirectory:   "/acme/local-env",
			WorkspaceFileName: DefaultWorkspaceFileName,
			Into:              "/acme",
		})

		assert.ErrorIs(t, err, ErrBranchNotFound)
		assert.Empty(t, git.cloned)
	})

	t.Run("branch without repo", func(t *testing.T) {
		m, git, _ := newManager()

		err := m.Initialise(context.Background(), InitialiseCommand{
			SourceDirectory:   "/acme/local-env",
			Branch:            "orca-adoption",
			WorkspaceFileName: DefaultWorkspaceFileName,
			Into:              "/acme",
		})

		assert.ErrorIs(t, err, ErrInvalidInput)
		assert.Empty(t, git.cloned)
	})
}

// registeringConfig records the workspaces added, then stops the init by
// saying they already exist.
type registeringConfig struct {
	config

	added []string
}

func (c *registeringConfig) AddWorkspace(path string, _ string) error {
	c.added = append(c.added, path)

	return ErrWorkspaceAlreadyExists
}

// GetWorkspaceMeta says the workspace is registered from somewhere else, so
// init stops there.
func (c *registeringConfig) GetWorkspaceMeta(name string) (WorkspaceMeta, error) {
	return WorkspaceMeta{Name: name, Path: "/elsewhere/orca.workspace.yaml"}, nil
}

func TestInitialiseFromLocalDirectory_RegistersAnAbsolutePath(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	fs := afero.NewMemMapFs()
	path := filepath.Join(wd, "local-env", DefaultWorkspaceFileName)
	require.NoError(t, afero.WriteFile(fs, path, []byte("name: acme\nprojects: []\n"), 0644))

	cfg := &registeringConfig{}
	m := NewManager(fs, stubTUI{}, cfg, &stubGit{}, nil, NewRepository(fs, nil), nil, nil)

	err = m.Initialise(context.Background(), InitialiseCommand{
		SourceDirectory:   "./local-env",
		WorkspaceFileName: DefaultWorkspaceFileName,
		Into:              "..",
	})

	assert.ErrorIs(t, err, ErrWorkspaceAlreadyExists)
	assert.Equal(t, []string{path}, cfg.added)
}

func TestCreatePreferences(t *testing.T) {
	cfg := &UnconfiguredWorkspace{
		Name:            "acme",
		PreferencesPath: "/acme/root/orca.preferences.yaml",
		Projects:        []UnconfiguredProject{{Name: "api"}, {Name: "web"}},
	}

	t.Run("lists every project", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		tui := &recordingTUI{}

		require.NoError(t, NewManager(fs, tui, nil, &stubGit{}, nil, nil, nil, nil).createPreferences(cfg))

		prefs, err := loadPreferences(fs, cfg.PreferencesPath)
		require.NoError(t, err)
		assert.Contains(t, prefs.Projects, "api")
		assert.Contains(t, prefs.Projects, "web")
		assert.Nil(t, prefs.Projects["api"].Enabled, "the workspace config decides")
		assert.Equal(t, []string{"Created /acme/root/orca.preferences.yaml, where you can enable and disable projects."}, tui.lines)
	})

	t.Run("keeps an existing file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, cfg.PreferencesPath, []byte("projects:\n  web:\n    enabled: false\n"), 0644))
		tui := &recordingTUI{}

		require.NoError(t, NewManager(fs, tui, nil, &stubGit{}, nil, nil, nil, nil).createPreferences(cfg))

		prefs, err := loadPreferences(fs, cfg.PreferencesPath)
		require.NoError(t, err)
		assert.False(t, *prefs.Projects["web"].Enabled)
		assert.Empty(t, tui.lines)
	})
}

func TestSuggestIgnoringPreferences(t *testing.T) {
	t.Run("suggests ignoring it", func(t *testing.T) {
		tui := &recordingTUI{}

		NewManager(afero.NewMemMapFs(), tui, nil, &stubGit{}, nil, nil, nil, nil).suggestIgnoringPreferences("/acme/root/orca.preferences.yaml")

		assert.Contains(t, tui.lines, "Note: /acme/root/orca.preferences.yaml holds your own preferences, like which projects are enabled.")
	})

	t.Run("not when it's already ignored", func(t *testing.T) {
		tui := &recordingTUI{}

		NewManager(afero.NewMemMapFs(), tui, nil, &stubGit{ignored: true}, nil, nil, nil, nil).suggestIgnoringPreferences("/acme/root/orca.preferences.yaml")

		assert.Empty(t, tui.lines)
	})
}

// initConfig is a user config that the workspace is registered in by init.
type initConfig struct {
	*cloneConfig

	added   []string
	current string
}

func (c *initConfig) AddWorkspace(path string, name string) error {
	if c.meta.Path != "" {
		return fmt.Errorf("%w: %s", ErrWorkspaceAlreadyExists, name)
	}

	c.meta = WorkspaceMeta{Name: name, Path: path}
	c.added = append(c.added, path)

	return nil
}

func (c *initConfig) GetCurrentWorkspace() string {
	return c.current
}

func TestInitialiseRunsHooks(t *testing.T) {
	const wsConfig = `
name: acme
projects:
  - name: local-env
    type: library
    repository:
      self: true
  - name: api
    repository:
      ssh: git@github.com:acme/api.git
`

	newManager := func(t *testing.T) (*Manager, *initConfig, *stubGit, *recordingHooks, *recordingTUI) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/acme/local-env/"+DefaultWorkspaceFileName, []byte(wsConfig), 0644))

		cfg := &initConfig{cloneConfig: &cloneConfig{}}
		git := &stubGit{fs: fs, root: "/acme/local-env", ignored: true, files: map[string]string{
			"/acme/api/" + DefaultProjectFileName: "hosts: [api.acme.test]\ntlsCerts: [\"*.acme.test\"]\n",
		}}
		hooks := &recordingHooks{}
		tui := &recordingTUI{}

		return NewManager(fs, tui, cfg, git, nil, NewRepository(fs, cfg), nil, hooks), cfg, git, hooks, tui
	}

	local := InitialiseCommand{SourceDirectory: "/acme/local-env", WorkspaceFileName: DefaultWorkspaceFileName, Into: "/acme"}

	t.Run("creates the preferences, clones, runs every hook, and shows what's next", func(t *testing.T) {
		m, cfg, git, hooks, tui := newManager(t)

		require.NoError(t, m.Initialise(context.Background(), local))

		assert.Equal(t, []string{"/acme/local-env/" + DefaultWorkspaceFileName}, cfg.added)
		assert.Equal(t, []string{"git@github.com:acme/api.git"}, git.cloned)
		assert.Equal(t, []SetUpOpts{{Projects: []string{"local-env", "api"}, Initialise: true}}, hooks.calls)

		prefs := slices.Index(tui.lines, "Created /acme/local-env/orca.preferences.yaml, where you can enable and disable projects.")
		cloning := slices.Index(tui.lines, "Cloning 'git@github.com:acme/api.git' into /acme/api...")
		assert.True(t, prefs >= 0 && prefs < cloning, "the preferences are created before anything's cloned")

		next := slices.Index(tui.lines, "Workspace acme is ready. Next steps:")
		require.GreaterOrEqual(t, next, 0)
		hosts := `sudo ORCA_CONFIG_PATH="$HOME/.orca/orca.yaml" orca hosts`
		step := func(cmd string, what string) string {
			return fmt.Sprintf("  %-*s   %s", len(hosts), cmd, what)
		}

		assert.Equal(t, []string{
			step("orca ws switch acme", "make it your current workspace"),
			step(hosts, "add its hosts to /etc/hosts"),
			step("orca tls gen", "generate its TLS certificates"),
			step("orca up", "start it"),
		}, tui.lines[next+1:])
	})

	t.Run("running it again clones nothing more, and runs the hooks again", func(t *testing.T) {
		m, cfg, git, hooks, tui := newManager(t)

		require.NoError(t, m.Initialise(context.Background(), local))
		require.NoError(t, m.Initialise(context.Background(), local))

		assert.Len(t, cfg.added, 1)
		assert.Equal(t, []string{"git@github.com:acme/api.git"}, git.cloned)
		assert.Len(t, hooks.calls, 2)
		assert.Equal(t, hooks.calls[0], hooks.calls[1])
		assert.Contains(t, tui.lines, "Workspace acme is already initialised here; cloning anything that's missing, and running its hooks again.")
	})

	t.Run("doesn't suggest switching to the current workspace", func(t *testing.T) {
		m, cfg, _, _, tui := newManager(t)
		cfg.current = "acme"

		require.NoError(t, m.Initialise(context.Background(), local))

		assert.NotContains(t, strings.Join(tui.lines, "\n"), "orca ws switch")
	})

	t.Run("another workspace with the same name", func(t *testing.T) {
		m, cfg, git, hooks, _ := newManager(t)
		cfg.meta = WorkspaceMeta{Name: "acme", Path: "/elsewhere/orca.workspace.yaml"}

		err := m.Initialise(context.Background(), local)

		assert.ErrorIs(t, err, ErrWorkspaceAlreadyExists)
		assert.ErrorContains(t, err, "There's already a workspace called acme, from /elsewhere/orca.workspace.yaml!")
		assert.Empty(t, git.cloned)
		assert.Empty(t, hooks.calls)
	})

	t.Run("skips the hooks, or runs them without asking", func(t *testing.T) {
		m, _, git, hooks, _ := newManager(t)
		skip := local
		skip.SkipHooks = true

		require.NoError(t, m.Initialise(context.Background(), skip))
		assert.Empty(t, hooks.calls)
		assert.Len(t, git.cloned, 1)

		yes := local
		yes.Yes = true

		require.NoError(t, m.Initialise(context.Background(), yes))
		assert.Equal(t, []SetUpOpts{{Projects: []string{"local-env", "api"}, Initialise: true, Yes: true}}, hooks.calls)
	})

	t.Run("from a repository that's already cloned, it's used as it is", func(t *testing.T) {
		m, _, git, hooks, tui := newManager(t)
		git.origins = map[string]string{"/acme/local-env": "git@github.com:acme/local-env.git"}
		m.githubClient = &stubGithub{configs: map[string]string{"": wsConfig}}

		err := m.Initialise(context.Background(), InitialiseCommand{SourceRepo: "acme/local-env", WorkspaceFileName: DefaultWorkspaceFileName, Into: "/acme"})

		require.NoError(t, err)
		assert.Equal(t, []string{"git@github.com:acme/api.git"}, git.cloned, "the workspace's repo isn't cloned again")
		assert.Contains(t, tui.lines, "acme/local-env is already cloned at /acme/local-env, so it's used as it is.")
		assert.Len(t, hooks.calls, 1)
	})

	t.Run("from a repository, when something else is in the way", func(t *testing.T) {
		m, _, git, hooks, _ := newManager(t)
		git.origins = map[string]string{"/acme/local-env": "git@github.com:someone/else.git"}
		m.githubClient = &stubGithub{configs: map[string]string{"": wsConfig}}

		err := m.Initialise(context.Background(), InitialiseCommand{SourceRepo: "acme/local-env", WorkspaceFileName: DefaultWorkspaceFileName, Into: "/acme"})

		assert.ErrorIs(t, err, ErrDirectoryAlreadyExists)
		assert.Empty(t, git.cloned)
		assert.Empty(t, hooks.calls)
	})
}
