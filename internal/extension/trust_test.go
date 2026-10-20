package extension

import (
	"context"
	"testing"

	orcatui "github.com/adamkirk/orca/internal/tui"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_TrustStore(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/home/me/.orca/trust.yaml"
	wsKey := TrustKey{Workspace: "acme", Extension: "seed"}
	projectKey := TrustKey{Workspace: "acme", Project: "api", Extension: "env"}

	t.Run("nothing is trusted to begin with", func(t *testing.T) {
		_, ok, err := NewTrustStore(fs, path).Get(wsKey)

		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("records extensions, which are read back by another store", func(t *testing.T) {
		require.NoError(t, NewTrustStore(fs, path).Trust(map[TrustKey]Trusted{
			wsKey:      {Fingerprint: "a", Summary: "./scripts/seed.sh"},
			projectKey: {Fingerprint: "b", Summary: ".env.example → .env"},
		}))

		store := NewTrustStore(fs, path)

		got, ok, err := store.Get(wsKey)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, Trusted{Fingerprint: "a", Summary: "./scripts/seed.sh"}, got)

		got, ok, err = store.Get(projectKey)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, Trusted{Fingerprint: "b", Summary: ".env.example → .env"}, got)

		// The same name in another project is another extension
		_, ok, err = store.Get(TrustKey{Workspace: "acme", Project: "web", Extension: "env"})
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("an invalid file", func(t *testing.T) {
		require.NoError(t, afero.WriteFile(fs, "/broken.yaml", []byte("workspaces: ["), 0600))

		_, _, err := NewTrustStore(fs, "/broken.yaml").Get(wsKey)

		assert.ErrorContains(t, err, "decoding /broken.yaml")
	})
}

// trustWorkspace has a workspace extension 'install-deps', used by api and web,
// and an 'env' in each of them.
func trustWorkspace() *workspace.Workspace {
	env := workspace.Extension{Name: "env", Spec: workspace.SyncFileSpec{Source: ".env.example", Target: ".env"}}

	return &workspace.Workspace{
		Name:       "acme",
		ConfigPath: "/code/acme/local-env/orca.workspace.yaml",
		Extensions: []workspace.Extension{
			command("install-deps", workspace.CommandSpec{Command: "npm ci"}),
			command("seed", workspace.CommandSpec{Command: "./scripts/seed.sh"}),
		},
		Projects: []workspace.Project{
			{Name: "api", ProjectDir: "/code/acme/api", IsRegistered: true, Config: workspace.ProjectConfig{Extensions: []workspace.Extension{env}}},
			{Name: "web", ProjectDir: "/code/acme/web", IsRegistered: true, Config: workspace.ProjectConfig{Extensions: []workspace.Extension{env}}},
		},
	}
}

// runs plans each project's 'env' and the shared 'install-deps' after it's
// cloned, then 'seed' after the workspace is initialised.
func runs(ws *workspace.Workspace) []planned {
	all := []planned{}

	for i := range ws.Projects {
		p := &ws.Projects[i]
		t := target{ws: ws, project: p}
		env, _ := p.FindExtension("env")
		deps, _ := ws.FindExtension("install-deps")

		all = append(all,
			planned{hook: "clone.post", owner: p, target: t, ext: env},
			planned{hook: "clone.post", target: t, ext: deps},
		)
	}

	seed, _ := ws.FindExtension("seed")

	return append(all, planned{hook: "init.post", target: target{ws: ws}, ext: seed})
}

func Test_Confirm(t *testing.T) {
	newManager := func(out *stubTui) (*Manager, *TrustStore) {
		store := newTrustStore()

		return NewManager(afero.NewMemMapFs(), nil, nil, out, store), store
	}

	t.Run("asks about each extension once, and remembers the answer", func(t *testing.T) {
		out := &stubTui{answer: true}
		m, store := newManager(out)
		ws := trustWorkspace()

		require.NoError(t, m.confirm(runs(ws), false))

		assert.Equal(t, 1, out.asked)
		assert.Equal(t, [][][]string{{
			{"api", "clone.post", "env", "syncFile", ".env.example → .env"},
			{"workspace", "clone.post", "install-deps", "command", "npm ci"},
			{"web", "clone.post", "env", "syncFile", ".env.example → .env"},
			{"workspace", "init.post", "seed", "command", "./scripts/seed.sh"},
		}}, out.tables)

		_, ok, err := store.Get(TrustKey{Workspace: "acme", Extension: "install-deps"})
		require.NoError(t, err)
		assert.True(t, ok)

		// Nothing has changed, so there's nothing to ask
		require.NoError(t, m.confirm(runs(ws), false))
		assert.Equal(t, 1, out.asked)
	})

	t.Run("only asks about an extension that's changed", func(t *testing.T) {
		out := &stubTui{answer: true}
		m, _ := newManager(out)
		ws := trustWorkspace()
		require.NoError(t, m.confirm(runs(ws), false))

		ws.Extensions[1] = command("seed", workspace.CommandSpec{Command: "./scripts/seed.sh", DefaultArgs: []string{"--fresh"}})
		out.tables = nil

		require.NoError(t, m.confirm(runs(ws), false))

		assert.Equal(t, 2, out.asked)
		assert.Equal(t, [][][]string{{
			{"workspace", "init.post", "seed", "command", "./scripts/seed.sh --fresh   (was: ./scripts/seed.sh)"},
		}}, out.tables)
		assert.Contains(t, out.lines, "These extensions have changed since you trusted them:")
	})

	t.Run("doesn't ask again when only the description changes", func(t *testing.T) {
		out := &stubTui{answer: true}
		m, _ := newManager(out)
		ws := trustWorkspace()
		require.NoError(t, m.confirm(runs(ws), false))

		ws.Extensions[1].Description = "Seed the databases"

		require.NoError(t, m.confirm(runs(ws), false))
		assert.Equal(t, 1, out.asked)
	})

	t.Run("declining records nothing", func(t *testing.T) {
		out := &stubTui{answer: false}
		m, store := newManager(out)

		err := m.confirm(runs(trustWorkspace()), false)

		assert.ErrorIs(t, err, ErrTrustDeclined)
		_, ok, err := store.Get(TrustKey{Workspace: "acme", Extension: "seed"})
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("yes doesn't ask, and records nothing", func(t *testing.T) {
		out := &stubTui{}
		m, store := newManager(out)

		require.NoError(t, m.confirm(runs(trustWorkspace()), true))

		assert.Zero(t, out.asked)
		_, ok, err := store.Get(TrustKey{Workspace: "acme", Extension: "seed"})
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("without a terminal, it suggests --yes", func(t *testing.T) {
		out := &stubTui{answerErr: orcatui.ErrNotInteractive}
		m, _ := newManager(out)

		err := m.confirm(runs(trustWorkspace()), false)

		assert.ErrorIs(t, err, ErrTrustNeeded)
		assert.ErrorIs(t, err, orcatui.ErrNotInteractive)
		assert.ErrorContains(t, err, "--yes")
	})
}

func Test_Run_TrustsTheExtension(t *testing.T) {
	ws := trustWorkspace()
	got := []ran{}
	out := &stubTui{answer: true}
	store := newTrustStore()
	m := NewManager(afero.NewMemMapFs(), nil, stubExecutor{ran: &got}, out, store)
	ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[0]})

	require.NoError(t, m.Run(ctx, RunCommand{Name: "install-deps"}))

	// It's the workspace's, so trusted for every project that uses it
	trusted, ok, err := store.Get(TrustKey{Workspace: "acme", Extension: "install-deps"})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "npm ci", trusted.Summary)

	planned := runs(ws)[:4]
	require.NoError(t, m.confirm(planned, false))
	assert.Equal(t, 1, out.asked, "only the envs are asked about")
	assert.Equal(t, [][][]string{{
		{"api", "clone.post", "env", "syncFile", ".env.example → .env"},
		{"web", "clone.post", "env", "syncFile", ".env.example → .env"},
	}}, out.tables)
}
