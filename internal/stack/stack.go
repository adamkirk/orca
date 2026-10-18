package stack

import (
	"github.com/adamkirk/orca/internal/workspace"
)

type tui interface {
	Info(msg ...string)
	Error(msg ...string)
	Success(msg ...string)
	NewLine()
	RecordIfError(msg string, err error) error
}

type compose interface {
	Up(*workspace.Workspace, *workspace.Project) error
	Down(ws *workspace.Workspace, p *workspace.Project) error
	ShowConfig(ws *workspace.Workspace, p *workspace.Project) error
	ShowCommand(ws *workspace.Workspace, p *workspace.Project) error
	Exec(ws *workspace.Workspace, p *workspace.Project, service string, cmdArgs []string) error
	Logs(ws *workspace.Workspace, p *workspace.Project, service string) error
	DoesSvcExist(ws *workspace.Workspace, p *workspace.Project, service string) (bool, error)
	IsSvcRunning(ws *workspace.Workspace, p *workspace.Project, service string) (bool, error)
	Run(ws *workspace.Workspace, p *workspace.Project, service string, cmdArgs []string) error
}

// Manager handles the commands that run a workspace or project's compose stack.
type Manager struct {
	compose compose
	tui     tui
}

func NewManager(compose compose, tui tui) *Manager {
	return &Manager{
		compose: compose,
		tui:     tui,
	}
}
