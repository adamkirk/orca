package workspace

import (
	"context"
	"os"
	"path/filepath"
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

		return NewManager(fs, stubTUI{}, nil, git, github, NewRepository(fs, nil), nil), git, github
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

func TestInitialiseFromLocalDirectory_RegistersAnAbsolutePath(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	fs := afero.NewMemMapFs()
	path := filepath.Join(wd, "local-env", DefaultWorkspaceFileName)
	require.NoError(t, afero.WriteFile(fs, path, []byte("name: acme\nprojects: []\n"), 0644))

	cfg := &registeringConfig{}
	m := NewManager(fs, stubTUI{}, cfg, &stubGit{}, nil, NewRepository(fs, nil), nil)

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

		require.NoError(t, NewManager(fs, tui, nil, &stubGit{}, nil, nil, nil).createPreferences(cfg))

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

		require.NoError(t, NewManager(fs, tui, nil, &stubGit{}, nil, nil, nil).createPreferences(cfg))

		prefs, err := loadPreferences(fs, cfg.PreferencesPath)
		require.NoError(t, err)
		assert.False(t, *prefs.Projects["web"].Enabled)
		assert.Empty(t, tui.lines)
	})
}

func TestSuggestIgnoringPreferences(t *testing.T) {
	t.Run("suggests ignoring it", func(t *testing.T) {
		tui := &recordingTUI{}

		NewManager(afero.NewMemMapFs(), tui, nil, &stubGit{}, nil, nil, nil).suggestIgnoringPreferences("/acme/root/orca.preferences.yaml")

		assert.Contains(t, tui.lines, "Next steps:")
	})

	t.Run("not when it's already ignored", func(t *testing.T) {
		tui := &recordingTUI{}

		NewManager(afero.NewMemMapFs(), tui, nil, &stubGit{ignored: true}, nil, nil, nil).suggestIgnoringPreferences("/acme/root/orca.preferences.yaml")

		assert.Empty(t, tui.lines)
	})
}
