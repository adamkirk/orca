package extension

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ran is a command that was run, on the host or in a service.
type ran struct {
	name    string
	args    []string
	dir     string
	service string
	exec    bool
}

type stubExecutor struct {
	ran *[]ran
	err error
}

func (e stubExecutor) Exec(name string, args []string, opts ...process.ExecOpt) error {
	cmd := exec.Command(name, args...)

	for _, opt := range opts {
		_ = opt(cmd)
	}

	*e.ran = append(*e.ran, ran{name: name, args: args, dir: cmd.Dir})

	return e.err
}

type stubCompose struct {
	ran      *[]ran
	services []string
	running  bool
}

func (c stubCompose) DoesSvcExist(_ *workspace.Workspace, _ *workspace.Project, service string) (bool, error) {
	for _, s := range c.services {
		if s == service {
			return true, nil
		}
	}

	return false, nil
}

func (c stubCompose) IsSvcRunning(*workspace.Workspace, *workspace.Project, string) (bool, error) {
	return c.running, nil
}

func (c stubCompose) Run(_ *workspace.Workspace, p *workspace.Project, service string, args []string) error {
	*c.ran = append(*c.ran, ran{name: args[0], args: args[1:], dir: p.ProjectDir, service: service})
	return nil
}

func (c stubCompose) Exec(_ *workspace.Workspace, p *workspace.Project, service string, args []string) error {
	*c.ran = append(*c.ran, ran{name: args[0], args: args[1:], dir: p.ProjectDir, service: service, exec: true})
	return nil
}

type stubTui struct {
	lines  []string
	tables [][][]string

	// The answer to every question, or the error asking it
	answer    bool
	answerErr error
	asked     int
}

func (t *stubTui) Info(msg ...string) {
	t.lines = append(t.lines, msg...)
}

func (t *stubTui) Table(_ []string, rows [][]string) {
	t.tables = append(t.tables, rows)
}

func (t *stubTui) Confirm(string) (bool, error) {
	t.asked++

	return t.answer, t.answerErr
}

func newTrustStore() *TrustStore {
	return NewTrustStore(afero.NewMemMapFs(), "/home/me/.orca/trust.yaml")
}

func command(name string, spec workspace.CommandSpec) workspace.Extension {
	return workspace.Extension{Name: name, Spec: spec}
}

// testWorkspace has a workspace extension 'seed', a service project 'api' that
// overrides the workspace's 'install', a library 'sdk', and 'web', which isn't
// cloned.
func testWorkspace() *workspace.Workspace {
	return &workspace.Workspace{
		Name:       "acme",
		ConfigPath: "/code/acme/local-env/orca.workspace.yaml",
		Extensions: []workspace.Extension{
			command("seed", workspace.CommandSpec{Command: "./scripts/seed.sh"}),
			command("install", workspace.CommandSpec{Command: "npm ci"}),
			command("migrate", workspace.CommandSpec{Command: "./bin/migrate", Service: "app"}),
		},
		Projects: []workspace.Project{
			{
				Name: "api", ProjectDir: "/code/acme/api", IsRegistered: true,
				Config: workspace.ProjectConfig{Extensions: []workspace.Extension{
					command("artisan", workspace.CommandSpec{Command: "php artisan", Service: "app", DefaultArgs: []string{"list"}}),
					command("install", workspace.CommandSpec{Command: "composer install", Chdir: "src"}),
				}},
			},
			{
				Name: "sdk", Type: workspace.ProjectTypeLibrary, ProjectDir: "/code/acme/api/packages/sdk", IsRegistered: true,
				Config: workspace.ProjectConfig{Extensions: []workspace.Extension{
					command("build", workspace.CommandSpec{Command: "make build"}),
				}},
			},
			{Name: "web"},
		},
	}
}

func Test_Run(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		cmd      RunCommand
		running  bool
		want     []ran
		wantInfo []string
		wantErr  error
	}{
		{
			name:    "a project's command in a service that isn't running",
			project: "api",
			cmd:     RunCommand{Name: "artisan"},
			want:    []ran{{name: "php", args: []string{"artisan", "list"}, dir: "/code/acme/api", service: "app"}},
		},
		{
			name:    "a project's command in a running service, with args",
			project: "api",
			cmd:     RunCommand{Name: "artisan", Args: []string{"migrate", "--force"}},
			running: true,
			want:    []ran{{name: "php", args: []string{"artisan", "migrate", "--force"}, dir: "/code/acme/api", service: "app", exec: true}},
		},
		{
			name:     "a project's own extension overrides the workspace's",
			project:  "api",
			cmd:      RunCommand{Name: "install"},
			want:     []ran{{name: "composer", args: []string{"install"}, dir: "/code/acme/api/src"}},
			wantInfo: []string{"Note: running api's own 'install', rather than the workspace's."},
		},
		{
			name:    "a workspace extension runs in the project's directory",
			project: "api",
			cmd:     RunCommand{Name: "seed"},
			want:    []ran{{name: "./scripts/seed.sh", args: []string{}, dir: "/code/acme/api"}},
		},
		{
			name: "a workspace extension, outside of any project, runs in the workspace's directory",
			cmd:  RunCommand{Name: "seed"},
			want: []ran{{name: "./scripts/seed.sh", args: []string{}, dir: "/code/acme/local-env"}},
		},
		{
			name:    "a library's extension runs on the host, in its directory",
			project: "sdk",
			cmd:     RunCommand{Name: "build"},
			want:    []ran{{name: "make", args: []string{"build"}, dir: "/code/acme/api/packages/sdk"}},
		},
		{
			name:    "a service command can't run in a library",
			project: "sdk",
			cmd:     RunCommand{Name: "migrate"},
			wantErr: ErrNoServices,
		},
		{
			name:    "a service command can't run outside of a project",
			cmd:     RunCommand{Name: "migrate"},
			wantErr: ErrNoServices,
		},
		{
			name:    "an unknown extension",
			project: "api",
			cmd:     RunCommand{Name: "deploy"},
			wantErr: workspace.ErrUnknownExtension,
		},
		{
			name:    "a project that isn't cloned",
			project: "web",
			cmd:     RunCommand{Name: "seed"},
			wantErr: ErrProjectNotCloned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := testWorkspace()
			execCtx := workspace.ExecutionContext{Workspace: ws}

			if tt.project != "" {
				p, err := ws.GetProject(tt.project)
				require.NoError(t, err)
				execCtx.Project = p
			}

			got := []ran{}
			out := &stubTui{}
			m := NewManager(afero.NewMemMapFs(), stubCompose{ran: &got, services: []string{"app"}, running: tt.running}, stubExecutor{ran: &got}, out, newTrustStore())

			err := m.Run(workspace.WithExecutionContext(context.Background(), execCtx), tt.cmd)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantInfo, out.lines)
		})
	}
}

func Test_Run_UnknownService(t *testing.T) {
	ws := testWorkspace()
	got := []ran{}
	m := NewManager(afero.NewMemMapFs(), stubCompose{ran: &got}, stubExecutor{ran: &got}, &stubTui{}, newTrustStore())
	ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[0]})

	err := m.Run(ctx, RunCommand{Name: "artisan"})

	assert.ErrorIs(t, err, ErrUnknownService)
	assert.Empty(t, got)
}

func Test_Run_CommandFails(t *testing.T) {
	ws := testWorkspace()
	failure := errors.New("exit status 1")
	got := []ran{}
	m := NewManager(afero.NewMemMapFs(), stubCompose{ran: &got}, stubExecutor{ran: &got, err: failure}, &stubTui{}, newTrustStore())
	ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws})

	err := m.Run(ctx, RunCommand{Name: "seed"})

	assert.ErrorIs(t, err, failure)

	msg, _, ok := usererr.Message(err)
	assert.True(t, ok)
	assert.Equal(t, "Extension 'seed' failed!", msg)
}

func Test_SyncFile(t *testing.T) {
	newFs := func(t *testing.T) afero.Fs {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/code/acme/api/.env.example", []byte("LOG_LEVEL=debug\n"), 0640))
		return fs
	}

	run := func(fs afero.Fs, spec workspace.SyncFileSpec, args ...string) (*stubTui, error) {
		ws := testWorkspace()
		out := &stubTui{}
		ws.Projects[0].Config.Extensions = []workspace.Extension{{Name: "env", Spec: spec}}
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[0]})

		return out, NewManager(fs, nil, nil, out, newTrustStore()).Run(ctx, RunCommand{Name: "env", Args: args})
	}

	t.Run("creates the target, keeping the source's permissions", func(t *testing.T) {
		fs := newFs(t)

		out, err := run(fs, workspace.SyncFileSpec{Source: ".env.example", Target: "config/.env"})

		require.NoError(t, err)
		contents, err := afero.ReadFile(fs, "/code/acme/api/config/.env")
		require.NoError(t, err)
		assert.Equal(t, "LOG_LEVEL=debug\n", string(contents))

		info, err := fs.Stat("/code/acme/api/config/.env")
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0640), info.Mode().Perm())
		assert.Equal(t, []string{"Created config/.env from .env.example."}, out.lines)
	})

	t.Run("never changes an existing target", func(t *testing.T) {
		fs := newFs(t)
		require.NoError(t, afero.WriteFile(fs, "/code/acme/api/.env", []byte("LOG_LEVEL=info\n"), 0600))

		out, err := run(fs, workspace.SyncFileSpec{Source: ".env.example", Target: ".env"})

		require.NoError(t, err)
		contents, err := afero.ReadFile(fs, "/code/acme/api/.env")
		require.NoError(t, err)
		assert.Equal(t, "LOG_LEVEL=info\n", string(contents))
		assert.Equal(t, []string{".env already exists, leaving it as it is."}, out.lines)
	})

	t.Run("a missing source", func(t *testing.T) {
		_, err := run(newFs(t), workspace.SyncFileSpec{Source: ".env.dist", Target: ".env"})

		assert.ErrorIs(t, err, ErrSourceNotFound)
	})

	t.Run("doesn't take arguments", func(t *testing.T) {
		_, err := run(newFs(t), workspace.SyncFileSpec{Source: ".env.example", Target: ".env"}, "--force")

		assert.ErrorIs(t, err, ErrUnexpectedArgs)
	})
}

func Test_Symlink(t *testing.T) {
	// Symlinks need a real filesystem
	setup := func(t *testing.T) (afero.Fs, *workspace.Workspace) {
		root := t.TempDir()
		ws := &workspace.Workspace{
			ConfigPath: filepath.Join(root, "local-env", "orca.workspace.yaml"),
			Projects: []workspace.Project{
				{Name: "web", ProjectDir: filepath.Join(root, "web"), IsRegistered: true},
				{Name: "sdk", Type: workspace.ProjectTypeLibrary, ProjectDir: filepath.Join(root, "sdk"), IsRegistered: true},
				{Name: "admin"},
			},
		}

		for _, p := range ws.Projects[:2] {
			require.NoError(t, os.MkdirAll(p.ProjectDir, 0755))
		}

		return afero.NewOsFs(), ws
	}

	run := func(fs afero.Fs, ws *workspace.Workspace, spec workspace.SymlinkSpec) error {
		ws.Projects[0].Config.Extensions = []workspace.Extension{{Name: "link", Spec: spec}}
		ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[0]})

		return NewManager(fs, nil, nil, &stubTui{}, newTrustStore()).Run(ctx, RunCommand{Name: "link"})
	}

	linkTo := func(t *testing.T, path string) string {
		dest, err := os.Readlink(path)
		require.NoError(t, err)
		return dest
	}

	spec := workspace.SymlinkSpec{Source: "{{ .Projects.sdk.Path }}", Target: "node_modules/@acme/sdk"}

	t.Run("links to another project, with an absolute link", func(t *testing.T) {
		fs, ws := setup(t)

		require.NoError(t, run(fs, ws, spec))
		assert.Equal(t, ws.Projects[1].ProjectDir, linkTo(t, filepath.Join(ws.Projects[0].ProjectDir, "node_modules/@acme/sdk")))
	})

	t.Run("a project with '-' in its name", func(t *testing.T) {
		fs, ws := setup(t)
		ws.Projects[1].Name = "acme-sdk"

		require.NoError(t, run(fs, ws, workspace.SymlinkSpec{Source: "{{ .Projects.acme-sdk.Path }}/dist", Target: "sdk"}))
		assert.Equal(t, ws.Projects[1].ProjectDir+"/dist", linkTo(t, filepath.Join(ws.Projects[0].ProjectDir, "sdk")))
	})

	t.Run("a relative source is made absolute", func(t *testing.T) {
		fs, ws := setup(t)

		require.NoError(t, run(fs, ws, workspace.SymlinkSpec{Source: "../sdk", Target: "sdk"}))
		assert.Equal(t, filepath.Join(ws.Projects[0].ProjectDir, "../sdk"), linkTo(t, filepath.Join(ws.Projects[0].ProjectDir, "sdk")))
	})

	t.Run("running again changes nothing", func(t *testing.T) {
		fs, ws := setup(t)

		require.NoError(t, run(fs, ws, spec))
		require.NoError(t, run(fs, ws, spec))
		assert.Equal(t, ws.Projects[1].ProjectDir, linkTo(t, filepath.Join(ws.Projects[0].ProjectDir, "node_modules/@acme/sdk")))
	})

	t.Run("doesn't replace anything else", func(t *testing.T) {
		fs, ws := setup(t)
		target := filepath.Join(ws.Projects[0].ProjectDir, "node_modules/@acme/sdk")
		require.NoError(t, os.MkdirAll(target, 0755))

		assert.ErrorIs(t, run(fs, ws, spec), ErrTargetExists)
		info, err := os.Lstat(target)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("doesn't replace a link to somewhere else", func(t *testing.T) {
		fs, ws := setup(t)
		target := filepath.Join(ws.Projects[0].ProjectDir, "sdk")
		require.NoError(t, os.Symlink("/elsewhere", target))

		assert.ErrorIs(t, run(fs, ws, workspace.SymlinkSpec{Source: "{{ .Projects.sdk.Path }}", Target: "sdk"}), ErrTargetExists)
		assert.Equal(t, "/elsewhere", linkTo(t, target))
	})

	t.Run("a project that isn't cloned", func(t *testing.T) {
		fs, ws := setup(t)

		assert.ErrorIs(t, run(fs, ws, workspace.SymlinkSpec{Source: "{{ .Projects.admin.Path }}", Target: "admin"}), ErrProjectNotCloned)
	})

	t.Run("a project that doesn't exist", func(t *testing.T) {
		fs, ws := setup(t)

		err := run(fs, ws, workspace.SymlinkSpec{Source: "{{ .Projects.nope.Path }}", Target: "nope"})

		assert.ErrorIs(t, err, workspace.ErrUnknownProject)
	})

	t.Run("a filesystem without symlinks", func(t *testing.T) {
		_, ws := setup(t)

		assert.ErrorIs(t, run(afero.NewMemMapFs(), ws, spec), ErrSymlinksNotSupported)
	})
}
