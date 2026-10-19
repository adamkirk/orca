package workspace

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

// stubGithub knows the branches of one repository, and the workspace config on
// each of them.
type stubGithub struct {
	configs map[string]string
	refs    []string
}

func (g *stubGithub) GetFileContents(owner string, repo string, path string, ref string) ([]byte, bool, error) {
	g.refs = append(g.refs, ref)

	contents, ok := g.configs[ref]

	return []byte(contents), ok, nil
}

func (g *stubGithub) BranchExists(owner string, repo string, branch string) (bool, error) {
	_, ok := g.configs[branch]

	return ok, nil
}

func TestInitialiseFromBranch(t *testing.T) {
	newManager := func() (*Manager, *stubGit, *stubGithub) {
		fs := afero.NewMemMapFs()
		git := &stubGit{
			branches: map[string][]string{
				"git@github.com:acme/api.git": {"main", "orca-adoption"},
				"git@github.com:acme/web.git": {"main"},
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

		return NewManager(fs, stubTUI{}, nil, git, github, NewRepository(fs, nil)), git, github
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
