package extension

import (
	"io"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/afero"
)

// compose runs commands in a project's compose services.
type compose interface {
	DoesSvcExist(ws *workspace.Workspace, p *workspace.Project, service string) (bool, error)
	IsSvcRunning(ws *workspace.Workspace, p *workspace.Project, service string) (bool, error)
	Run(ws *workspace.Workspace, p *workspace.Project, service string, cmdArgs []string, output io.Writer) error
	Exec(ws *workspace.Workspace, p *workspace.Project, service string, cmdArgs []string, output io.Writer) error
}

// executor runs commands on the host.
type executor interface {
	Exec(cmdName string, args []string, opts ...process.ExecOpt) error
}

type tui interface {
	Info(msg ...string)
	Indented(prefix string) io.Writer
	Table(header []string, rows [][]string)
	Confirm(question string) (bool, error)
}

// trustStore records which extensions the developer has trusted to run.
type trustStore interface {
	Get(key TrustKey) (Trusted, bool, error)
	Trust(trusted map[TrustKey]Trusted) error
}

// Manager runs extensions, the named steps defined in workspace and project
// configs.
type Manager struct {
	fs       afero.Fs
	compose  compose
	executor executor
	tui      tui
	trust    trustStore
}

func NewManager(fs afero.Fs, compose compose, executor executor, tui tui, trust trustStore) *Manager {
	return &Manager{
		fs:       fs,
		compose:  compose,
		executor: executor,
		tui:      tui,
		trust:    trust,
	}
}
