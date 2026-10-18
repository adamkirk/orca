package controller

import (
	"github.com/adamkirk/orca/internal/common"
)

type config interface {
	GetAllProjectMeta() []common.ProjectMeta
	GetCurrentWorkspace() string
	GetWorkspaceMeta(name string) (common.WorkspaceMeta, error)
}

type tui interface {
	Info(msg ...string)
	Error(msg ...string)
	Success(msg ...string)
	NewLine()
	RecordIfError(msg string, err error) error
}

type etcHostsManager interface {
	SyncForWorkspace(ws *common.Workspace) error
}

type workspaceRepository interface {
	Load(name string) (*common.Workspace, error)
}

type compose interface {
	Up(*common.Workspace, *common.Project) error
	Down(ws *common.Workspace, p *common.Project) error
	ShowConfig(ws *common.Workspace, p *common.Project) error
	ShowCommand(ws *common.Workspace, p *common.Project) error
	Exec(ws *common.Workspace, p *common.Project, service string, cmdArgs []string) error
	Logs(ws *common.Workspace, p *common.Project, service string) error
	DoesSvcExist(ws *common.Workspace, p *common.Project, service string) (bool, error)
	IsSvcRunning(ws *common.Workspace, p *common.Project, service string) (bool, error)
	Run(ws *common.Workspace, p *common.Project, service string, cmdArgs []string) error
}

type provisionerRunner interface {
	RunAll(ws *common.Workspace, p *common.Project) error
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
