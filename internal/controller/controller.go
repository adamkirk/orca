package controller

import (
	"github.com/adamkirk/orca/internal/workspace"
)

type config interface {
	GetAllProjectMeta() []workspace.ProjectMeta
	GetCurrentWorkspace() string
	GetWorkspaceMeta(name string) (workspace.WorkspaceMeta, error)
}

type tui interface {
	Info(msg ...string)
	Error(msg ...string)
	Success(msg ...string)
	NewLine()
	RecordIfError(msg string, err error) error
}

type etcHostsManager interface {
	SyncForWorkspace(ws *workspace.Workspace) error
}

type workspaceRepository interface {
	Load(name string) (*workspace.Workspace, error)
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

type provisionerRunner interface {
	RunAll(ws *workspace.Workspace, p *workspace.Project) error
}

type Controller struct {
	cfg               config
	workspaceRepo     workspaceRepository
	compose           compose
	tui               tui
	etcHostsManager   etcHostsManager
	provisionerRunner provisionerRunner
}

func NewController(
	cfg config,
	wsRepo workspaceRepository,
	compose compose,
	tui tui,
	etcHostsManager etcHostsManager,
	provisionerRunner provisionerRunner,
) *Controller {
	return &Controller{
		cfg:               cfg,
		workspaceRepo:     wsRepo,
		compose:           compose,
		tui:               tui,
		etcHostsManager:   etcHostsManager,
		provisionerRunner: provisionerRunner,
	}
}
