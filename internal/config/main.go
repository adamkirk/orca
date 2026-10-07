package config

import (
	"path/filepath"

	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/pkg/slices"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// Config is responsible for loading the configuration and managing it on disk.
// As a general rule whenever we're reading values we should be working with the
// runtimeConfig instance. Outside, in the main package, we should be using
// viper to override certain variables for the runtimeConfig with their corresponding
// CLI argument or environment variable equivalents. For any mutation methods, we
// generally should be working with the persisted config. This is too ensure that
// if the user has for example, overridden the logging level for a single command,
// we don't persist that log level permanently in the config on disk.
// This can be a bit finicky in practice, but takes a little care to think here
// about what can and cannot be overridden at the CLI or ENV levels. If anything
// changes in here, that cannot be overridden at those levels we should also reflect
// those changes for the runtimeConfig.
type Config struct {
	// Not exported, for internal use
	fs            afero.Fs
	persisted     *config
	persistedPath string

	runtimeConfig *config
}

func (self *Config) save() error {
	contents, err := yaml.Marshal(self.persisted)
	if err != nil {
		return err
	}

	dir := filepath.Dir(self.persistedPath)

	if err := self.fs.MkdirAll(dir, 0755); err != nil {
		return err
	}

	err = afero.WriteFile(self.fs, self.persistedPath, contents, 0655)

	if err != nil {
		return err
	}

	return nil
}

// SwitchWorkspace changes the current workspace in the configuration on disk. As
// this can be overridden by the CLI arguments or environment variables, we don't
// update the inUse configuration.
func (self *Config) SwitchWorkspace(workspaceName string) error {
	if !self.persisted.workspaceExists(workspaceName) {
		return common.ErrUnknownWorkspace{
			Name: workspaceName,
		}
	}

	self.persisted.CurrentWorkspace = workspaceName

	return self.save()
}

// ClearCurrentWorkspace deselects any workspace on a global level.
func (self *Config) ClearCurrentWorkspace() error {
	self.persisted.CurrentWorkspace = ""

	return self.save()
}

// AddWorkspace adds a new workspace to the configuration on disk. In the case of
// adding a new workspace it should also update the workspaces for the inUse
// config as it cannot be overridden by any CLI arguments or environment variables.
func (self *Config) AddWorkspace(path string, name string) error {
	if self.persisted.workspaceExists(name) {
		return common.ErrWorkspaceAlreadyExists{
			Name: name,
		}
	}

	self.persisted.Workspaces = append(self.persisted.Workspaces, configWorkspace{
		Name: name,
		Path: path,
	})

	self.runtimeConfig.Workspaces = self.persisted.Workspaces

	return self.save()
}

func (self *Config) GetAllProjectMeta() []common.ProjectMeta {
	projects := []common.ProjectMeta{}

	for _, ws := range self.runtimeConfig.Workspaces {
		for _, p := range ws.Projects {
			projects = append(projects, common.ProjectMeta{
				Name:          p.Name,
				WorkspaceName: ws.Name,
				Path:          p.Path,
			})
		}
	}

	return projects
}

func (self *Config) GetProjectMeta(wsName string, projectName string) (common.ProjectMeta, error) {
	ws, err := self.runtimeConfig.getWorkspace(wsName)

	if err != nil {
		return common.ProjectMeta{}, err
	}

	p := slices.GetNamedElement(ws.Projects, projectName)

	if p == nil {
		return common.ProjectMeta{}, common.ErrUnknownProject{
			Name: projectName,
		}
	}

	return common.ProjectMeta{
		WorkspaceName: ws.Name,
		Name:          p.Name,
		Path:          p.Path,
	}, nil
}

// GetWorkspaceLocation returns a tuple with name and path for the given namespace
// if it does in fact exist. If it doesn't exist an error is returned.
func (self *Config) GetWorkspaceMeta(name string) (common.WorkspaceMeta, error) {
	v := slices.GetNamedElement(self.persisted.Workspaces, name)

	if v == nil {
		return common.WorkspaceMeta{}, common.ErrUnknownWorkspace{
			Name: name,
		}
	}

	return common.WorkspaceMeta{
		Name: v.Name,
		Path: v.Path,
	}, nil
}

// GetWorkspaceLocations returns a tuple with the name and path for each workspace.
func (self *Config) GetAllWorkspaceMeta() []common.WorkspaceMeta {
	locs := make([]common.WorkspaceMeta, len(self.runtimeConfig.Workspaces))

	for i, ws := range self.runtimeConfig.Workspaces {
		locs[i] = common.WorkspaceMeta{
			Name: ws.Name,
			Path: ws.Path,
		}
	}

	return locs
}

func (self *Config) ProjectExists(wsName string, name string) (bool, error) {
	return self.runtimeConfig.projectExists(wsName, name)
}

func (self *Config) SetProjectPath(wsName string, name string, path string) error {
	ws, err := self.persisted.getWorkspace(wsName)
	if err != nil {
		return err
	}

	project, err := self.persisted.getProject(wsName, name)

	if err != nil {
		if _, ok := err.(common.ErrUnknownProject); !ok {
			return err
		}

		project = configProject{
			Name: name,
		}
	}

	project.Path = path

	ws.Projects = slices.UpsertNamedElement(ws.Projects, project)
	self.persisted.Workspaces = slices.UpsertNamedElement(self.persisted.Workspaces, ws)

	self.runtimeConfig.Workspaces = self.persisted.Workspaces

	return self.save()
}

func (self *Config) GetLoggingLevel() string {
	return self.runtimeConfig.Logging.Level
}

func (self *Config) GetLoggingFormat() string {
	return self.runtimeConfig.Logging.Format
}

func (self *Config) GetPluginDirs() []string {
	return self.runtimeConfig.Plugins.Dirs
}

func (self *Config) GetRuntimeConfig() *config {
	return self.runtimeConfig
}

// TODO: add a test
func (self *Config) GetCurrentWorkspace() string {
	return self.runtimeConfig.CurrentWorkspace
}

func (self *Config) loadFromFile() (config, error) {
	contents, err := afero.ReadFile(self.fs, self.persistedPath)

	if err != nil {
		return config{}, err
	}

	cfg := buildDefaultConfig()
	err = yaml.Unmarshal(contents, cfg)

	return *cfg, err
}

func (self *Config) LoadOrCreate() error {
	exists, err := afero.Exists(self.fs, self.persistedPath)

	if err != nil {
		return err
	}

	if !exists {
		self.persisted = buildDefaultConfig()
		self.runtimeConfig = buildDefaultConfig()

		return self.save()
	}

	cfg, err := self.loadFromFile()
	// Ensure we don't end up pointing at the same thing, to avoid confusion when
	// writing the config, that may override temporary changes applied by CLI args
	// or env vars
	persisted := cfg

	self.runtimeConfig = &persisted
	self.persisted = &cfg

	return err
}

func buildDefaultConfig() *config {
	return &config{
		Logging: configLogging{
			Level:  "none",
			Format: "text",
		},
		Workspaces:       []configWorkspace{},
		CurrentWorkspace: "",
	}
}

func NewDefaultConfig(fs afero.Fs, path string) *Config {
	return &Config{
		fs:            fs,
		persisted:     nil,
		persistedPath: path,

		runtimeConfig: nil,
	}
}
