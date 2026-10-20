package workspace

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type stubTUI struct{}

func (stubTUI) Info(msg ...string)                     {}
func (stubTUI) Error(msg ...string)                    {}
func (stubTUI) Success(msg ...string)                  {}
func (stubTUI) Table(header []string, rows [][]string) {}

// stubGit knows the branches on each remote, keyed by URL, and records
// clones.
type stubGit struct {
	branches map[string][]string
	checked  []string
	cloned   []string
	ignored  bool

	// Clones create their directory in fs, when it's set
	fs afero.Fs
}

func (g *stubGit) GetRepositoryRootFromPath(path string) (string, error) {
	return "", nil
}

func (g *stubGit) Clone(repoUrl string, target string, branch string) error {
	g.cloned = append(g.cloned, repoUrl)

	if g.fs != nil {
		return g.fs.MkdirAll(target, 0755)
	}

	return nil
}

func (g *stubGit) RemoteBranchExists(repoUrl string, branch string) (bool, error) {
	g.checked = append(g.checked, repoUrl)

	for _, b := range g.branches[repoUrl] {
		if b == branch {
			return true, nil
		}
	}

	return false, nil
}

func (g *stubGit) DirectoryHasOrigin(dir string, origin string) (bool, error) {
	return false, nil
}

func (g *stubGit) IsIgnored(path string) (bool, error) {
	return g.ignored, nil
}

func TestCheckBranches(t *testing.T) {
	git := &stubGit{
		branches: map[string][]string{
			"git@github.com:acme/api.git": {"main", "orca-adoption"},
			"git@github.com:acme/web.git": {"main"},
			"git@github.com:acme/db.git":  {"main"},
		},
	}

	fs := afero.NewMemMapFs()
	assert.NoError(t, fs.MkdirAll("/acme/db", 0755))

	m := NewManager(fs, stubTUI{}, nil, git, nil, nil, nil)

	target := func(name string, repo ProjectRepositoryConfig) cloneTarget {
		return cloneTarget{project: Project{Name: name, RepositoryConfig: repo}, into: "/acme/" + name}
	}

	tests := []struct {
		name        string
		targets     []cloneTarget
		wantChecked []string
		wantErr     string
	}{
		{
			name: "all branches exist",
			targets: []cloneTarget{
				target("api", ProjectRepositoryConfig{SSH: "git@github.com:acme/api.git", Branch: "orca-adoption"}),
				target("web", ProjectRepositoryConfig{SSH: "git@github.com:acme/web.git", Branch: "main"}),
			},
			wantChecked: []string{"git@github.com:acme/api.git", "git@github.com:acme/web.git"},
		},
		{
			name: "every missing branch is listed",
			targets: []cloneTarget{
				target("api", ProjectRepositoryConfig{SSH: "git@github.com:acme/api.git", Branch: "typo"}),
				target("web", ProjectRepositoryConfig{SSH: "git@github.com:acme/web.git", Branch: "orca-adoption"}),
			},
			wantChecked: []string{"git@github.com:acme/api.git", "git@github.com:acme/web.git"},
			wantErr:     "branch not found: api: 'typo' (git@github.com:acme/api.git), web: 'orca-adoption' (git@github.com:acme/web.git)",
		},
		{
			name: "projects that won't be cloned are skipped",
			targets: []cloneTarget{
				target("root", ProjectRepositoryConfig{Self: true}),
				target("api", ProjectRepositoryConfig{SSH: "git@github.com:acme/api.git"}),
				target("db", ProjectRepositoryConfig{SSH: "git@github.com:acme/db.git", Branch: "orca-adoption"}),
			},
			wantChecked: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git.checked = nil

			err := m.checkBranches(tt.targets)

			assert.Equal(t, tt.wantChecked, git.checked)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			assert.ErrorIs(t, err, ErrBranchNotFound)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// cloneConfig registers the one workspace, and records the projects cloned.
type cloneConfig struct {
	config

	meta       WorkspaceMeta
	registered []string
	metas      []ProjectMeta
}

func (c *cloneConfig) GetWorkspaceMeta(name string) (WorkspaceMeta, error) {
	return c.meta, nil
}

func (c *cloneConfig) ProjectExists(wsName string, name string) (bool, error) {
	return false, nil
}

func (c *cloneConfig) RegisterProject(wsName string, meta ProjectMeta) error {
	c.registered = append(c.registered, meta.Name)
	c.metas = append(c.metas, meta)

	return nil
}

func TestCloneIncludesDisabledProjects(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/acme/root/orca.workspace.yaml"

	assert.NoError(t, afero.WriteFile(fs, path, []byte(`
name: acme
projects:
  - name: api
    repository:
      ssh: git@github.com:acme/api.git
  - name: web
    repository:
      ssh: git@github.com:acme/web.git
`), 0644))
	assert.NoError(t, afero.WriteFile(fs, "/acme/root/"+DefaultPreferencesFileName, []byte("projects:\n  web:\n    enabled: false\n"), 0644))

	meta := WorkspaceMeta{Name: "acme", Path: path}
	cfg := &cloneConfig{meta: meta}
	git := &stubGit{}
	repo := NewRepository(fs, stubUserConfig{workspaces: map[string]WorkspaceMeta{"acme": meta}})

	err := NewManager(fs, stubTUI{}, cfg, git, nil, repo, nil).clone("acme", "", "/acme", true)

	assert.NoError(t, err)
	assert.Equal(t, []string{"git@github.com:acme/api.git", "git@github.com:acme/web.git"}, git.cloned)
	assert.Equal(t, []string{"api", "web"}, cfg.registered)
}

func TestCloneOrder(t *testing.T) {
	target := func(name string, into string, requires ...string) cloneTarget {
		return cloneTarget{
			project: Project{Name: name, Requires: requires, RepositoryConfig: ProjectRepositoryConfig{SSH: name}},
			into:    into,
		}
	}

	self := cloneTarget{project: Project{Name: "root", RepositoryConfig: ProjectRepositoryConfig{Self: true}}, into: "/acme"}

	tests := []struct {
		name    string
		targets []cloneTarget
		want    []string
		wantErr error
	}{
		{
			name:    "keeps the configured order",
			targets: []cloneTarget{target("web", "/acme/web"), target("api", "/acme/api")},
			want:    []string{"web", "api"},
		},
		{
			name:    "requirements first",
			targets: []cloneTarget{target("web", "/acme/web", "sdk"), target("sdk", "/acme/sdk")},
			want:    []string{"sdk", "web"},
		},
		{
			name:    "a project inside another after it",
			targets: []cloneTarget{target("sdk", "/acme/api/packages/sdk"), target("api", "/acme/api")},
			want:    []string{"api", "sdk"},
		},
		{
			name:    "a similar name isn't inside",
			targets: []cloneTarget{target("api-v2", "/acme/api-v2"), target("api", "/acme/api")},
			want:    []string{"api-v2", "api"},
		},
		{
			name:    "the workspace's own repository encloses nothing",
			targets: []cloneTarget{target("api", "/acme/api"), self},
			want:    []string{"api", "root"},
		},
		{
			name: "both together",
			targets: []cloneTarget{
				target("web", "/acme/web", "sdk", "unknown"),
				target("sdk", "/acme/api/packages/sdk"),
				target("api", "/acme/api"),
			},
			want: []string{"api", "sdk", "web"},
		},
		{
			name:    "a circle fails",
			targets: []cloneTarget{target("api", "/acme/api", "sdk"), target("sdk", "/acme/api/sdk")},
			wantErr: ErrCircularCloneOrder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ordered, err := cloneOrder(tt.targets)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			assert.NoError(t, err)

			names := []string{}

			for _, o := range ordered {
				names = append(names, o.project.Name)
			}

			assert.Equal(t, tt.want, names)
		})
	}
}

func TestCloneLibraryIntoPath(t *testing.T) {
	newManager := func(t *testing.T, ignored bool) (*Manager, *cloneConfig, *stubGit, *recordingTUI) {
		fs := afero.NewMemMapFs()
		path := "/acme/root/orca.workspace.yaml"

		assert.NoError(t, afero.WriteFile(fs, path, []byte(`
name: acme
projects:
  - name: web
    repository:
      ssh: git@github.com:acme/web.git
    requires:
      - sdk
  - name: sdk
    type: library
    path: api/packages/sdk
    repository:
      ssh: git@github.com:acme/sdk.git
  - name: api
    repository:
      ssh: git@github.com:acme/api.git
`), 0644))

		meta := WorkspaceMeta{Name: "acme", Path: path}
		cfg := &cloneConfig{meta: meta}
		git := &stubGit{fs: fs, ignored: ignored}
		tui := &recordingTUI{}
		repo := NewRepository(fs, stubUserConfig{workspaces: map[string]WorkspaceMeta{"acme": meta}})

		return NewManager(fs, tui, cfg, git, nil, repo, nil), cfg, git, tui
	}

	warning := "Warning: sdk is inside api, whose repository doesn't ignore packages/sdk, so it shows up there as untracked files. Add it to api's .gitignore."

	t.Run("clones it after its parent and before what requires it, and registers its type", func(t *testing.T) {
		m, cfg, git, tui := newManager(t, true)

		assert.NoError(t, m.clone("acme", "", "/acme", true))

		assert.Equal(t, []string{"git@github.com:acme/api.git", "git@github.com:acme/sdk.git", "git@github.com:acme/web.git"}, git.cloned)
		assert.Equal(t, []ProjectMeta{
			{Name: "api", WorkspaceName: "acme", Path: "/acme/api", Type: ProjectTypeService},
			{Name: "sdk", WorkspaceName: "acme", Path: "/acme/api/packages/sdk", Type: ProjectTypeLibrary},
			{Name: "web", WorkspaceName: "acme", Path: "/acme/web", Type: ProjectTypeService},
		}, cfg.metas)
		assert.Contains(t, tui.lines, "Cloning 'git@github.com:acme/sdk.git' (library) into /acme/api/packages/sdk...")
		assert.NotContains(t, tui.lines, warning)
	})

	t.Run("warns when the parent repository doesn't ignore it", func(t *testing.T) {
		m, _, _, tui := newManager(t, false)

		assert.NoError(t, m.clone("acme", "", "/acme", true))

		assert.Contains(t, tui.lines, warning)
	})

	t.Run("cloning it on its own needs its parent cloned", func(t *testing.T) {
		m, cfg, git, _ := newManager(t, true)

		err := m.clone("acme", "sdk", "", true)

		assert.ErrorIs(t, err, ErrEnclosingProjectNotCloned)
		assert.Empty(t, git.cloned)
		assert.Empty(t, cfg.registered)
	})
}
