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
}

func (g *stubGit) GetRepositoryRootFromPath(path string) (string, error) {
	return "", nil
}

func (g *stubGit) Clone(repoUrl string, target string, branch string) error {
	g.cloned = append(g.cloned, repoUrl)

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

	m := NewManager(fs, stubTUI{}, nil, git, nil, nil)

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
