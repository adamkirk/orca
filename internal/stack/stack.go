package stack

import (
	"io"

	"github.com/adamkirk/orca/internal/workspace"
)

type compose interface {
	Up(*workspace.Workspace, *workspace.Project) error
	Down(ws *workspace.Workspace, p *workspace.Project) error
	ShowConfig(ws *workspace.Workspace, p *workspace.Project) error
	ShowCommand(ws *workspace.Workspace, p *workspace.Project) error
	Exec(ws *workspace.Workspace, p *workspace.Project, service string, cmdArgs []string, output io.Writer) error
	Logs(ws *workspace.Workspace, p *workspace.Project, service string) error
	DoesSvcExist(ws *workspace.Workspace, p *workspace.Project, service string) (bool, error)
	IsSvcRunning(ws *workspace.Workspace, p *workspace.Project, service string) (bool, error)
	Run(ws *workspace.Workspace, p *workspace.Project, service string, cmdArgs []string, output io.Writer) error
	ServiceNames(p *workspace.Project) ([]string, error)
	IsProjectRunning(ws *workspace.Workspace, p *workspace.Project) (bool, error)
}

type tui interface {
	Info(msg ...string)
}

// hooks runs the hooks around starting and stopping projects.
type hooks interface {
	ConfirmHooks(ws *workspace.Workspace, runs []workspace.HookRun, yes bool) error
	RunConfirmedHook(ws *workspace.Workspace, run workspace.HookRun) error
}

// Manager handles the commands that run a workspace or project's compose stack.
type Manager struct {
	compose compose
	tui     tui
	hooks   hooks
}

func NewManager(compose compose, tui tui, hooks hooks) *Manager {
	return &Manager{
		compose: compose,
		tui:     tui,
		hooks:   hooks,
	}
}
