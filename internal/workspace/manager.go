package workspace

import (
	"github.com/spf13/afero"
)

type tui interface {
	Info(msg ...string)
	Error(msg ...string)
	Success(msg ...string)
	RecordIfError(msg string, err error) error
	Table(header []string, rows [][]string)
}

type config interface {
	AddWorkspace(configFilePath string, name string) error
	GetAllWorkspaceMeta() []WorkspaceMeta
	GetCurrentWorkspace() string
	GetWorkspaceMeta(name string) (WorkspaceMeta, error)
	SetProjectPath(wsName string, name string, into string) error
	ProjectExists(wsName string, name string) (bool, error)
	ClearCurrentWorkspace() error
	SwitchWorkspace(workspaceName string) error
}

type git interface {
	GetRepositoryRootFromPath(path string) (string, error)
	Clone(repoUrl string, target string, dir string) error
	DirectoryHasOrigin(dir string, origin string) (bool, error)
}

type githubClient interface {
	GetFileContents(owner string, repo string, path string) ([]byte, bool, error)
}

type workspaceRepo interface {
	Load(name string) (*Workspace, error)
	LoadUnconfiguredWorkspace(path string) (*UnconfiguredWorkspace, error)
	ParseUnconfiguredWorkspace(contents []byte, path string) (*UnconfiguredWorkspace, error)
}

type Manager struct {
	fs            afero.Fs
	tui           tui
	configManager config
	git           git
	githubClient  githubClient
	workspaceRepo workspaceRepo
}

func NewManager(fs afero.Fs, tui tui, configManager config, git git, githubClient githubClient, workspaceRepo workspaceRepo) *Manager {
	return &Manager{
		fs:            fs,
		tui:           tui,
		configManager: configManager,
		git:           git,
		githubClient:  githubClient,
		workspaceRepo: workspaceRepo,
	}
}
