package config

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/adamkirk/orca/internal/workspace"
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

func (c *Config) save() error {
	contents, err := yaml.Marshal(c.persisted)
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.persistedPath)

	if err := c.fs.MkdirAll(dir, 0755); err != nil {
		return err
	}

	err = afero.WriteFile(c.fs, c.persistedPath, contents, 0655)

	if err != nil {
		return err
	}

	return nil
}

// SwitchWorkspace changes the current workspace in the configuration on disk. As
// this can be overridden by the CLI arguments or environment variables, we don't
// update the inUse configuration.
func (c *Config) SwitchWorkspace(workspaceName string) error {
	if !c.persisted.workspaceExists(workspaceName) {
		return fmt.Errorf("%w: %s", workspace.ErrUnknownWorkspace, workspaceName)
	}

	c.persisted.CurrentWorkspace = workspaceName

	return c.save()
}

// ClearCurrentWorkspace deselects any workspace on a global level.
func (c *Config) ClearCurrentWorkspace() error {
	c.persisted.CurrentWorkspace = ""

	return c.save()
}

// AddWorkspace adds a new workspace to the configuration on disk. In the case of
// adding a new workspace it should also update the workspaces for the inUse
// config as it cannot be overridden by any CLI arguments or environment variables.
func (c *Config) AddWorkspace(path string, name string) error {
	if c.persisted.workspaceExists(name) {
		return fmt.Errorf("%w: %s", workspace.ErrWorkspaceAlreadyExists, name)
	}

	c.persisted.Workspaces = append(c.persisted.Workspaces, configWorkspace{
		Name: name,
		Path: path,
	})

	c.runtimeConfig.Workspaces = c.persisted.Workspaces

	return c.save()
}

func (c *Config) GetAllProjectMeta() []workspace.ProjectMeta {
	projects := []workspace.ProjectMeta{}

	for _, ws := range c.runtimeConfig.Workspaces {
		for _, p := range ws.Projects {
			projects = append(projects, workspace.ProjectMeta{
				Name:          p.Name,
				WorkspaceName: ws.Name,
				Path:          p.Path,
				Type:          workspace.ProjectType(p.Type),
			})
		}
	}

	return projects
}

func (c *Config) GetProjectMeta(wsName string, projectName string) (workspace.ProjectMeta, error) {
	ws, err := c.runtimeConfig.getWorkspace(wsName)

	if err != nil {
		return workspace.ProjectMeta{}, err
	}

	i := ws.projectIndex(projectName)

	if i == -1 {
		return workspace.ProjectMeta{}, fmt.Errorf("%w: %s", workspace.ErrUnknownProject, projectName)
	}

	return workspace.ProjectMeta{
		WorkspaceName: ws.Name,
		Name:          ws.Projects[i].Name,
		Path:          ws.Projects[i].Path,
		Type:          workspace.ProjectType(ws.Projects[i].Type),
	}, nil
}

// GetWorkspaceLocation returns a tuple with name and path for the given namespace
// if it does in fact exist. If it doesn't exist an error is returned.
func (c *Config) GetWorkspaceMeta(name string) (workspace.WorkspaceMeta, error) {
	i := c.persisted.workspaceIndex(name)

	if i == -1 {
		return workspace.WorkspaceMeta{}, fmt.Errorf("%w: %s", workspace.ErrUnknownWorkspace, name)
	}

	return workspace.WorkspaceMeta{
		Name: c.persisted.Workspaces[i].Name,
		Path: c.persisted.Workspaces[i].Path,
	}, nil
}

// GetWorkspaceLocations returns a tuple with the name and path for each workspace.
func (c *Config) GetAllWorkspaceMeta() []workspace.WorkspaceMeta {
	locs := make([]workspace.WorkspaceMeta, len(c.runtimeConfig.Workspaces))

	for i, ws := range c.runtimeConfig.Workspaces {
		locs[i] = workspace.WorkspaceMeta{
			Name: ws.Name,
			Path: ws.Path,
		}
	}

	return locs
}

func (c *Config) ProjectExists(wsName string, name string) (bool, error) {
	return c.runtimeConfig.projectExists(wsName, name)
}

// RegisterProject records where a project of the workspace is, and its type,
// replacing anything recorded for it before.
func (c *Config) RegisterProject(wsName string, meta workspace.ProjectMeta) error {
	ws, err := c.persisted.getWorkspace(wsName)
	if err != nil {
		return err
	}

	project, err := c.persisted.getProject(wsName, meta.Name)

	if err != nil {
		if !errors.Is(err, workspace.ErrUnknownProject) {
			return err
		}

		project = configProject{
			Name: meta.Name,
		}
	}

	project.Path = meta.Path
	project.Type = string(meta.Type)

	ws.Projects = upsert(ws.Projects, ws.projectIndex(project.Name), project)
	c.persisted.Workspaces = upsert(c.persisted.Workspaces, c.persisted.workspaceIndex(ws.Name), ws)

	c.runtimeConfig.Workspaces = c.persisted.Workspaces

	return c.save()
}

func (c *Config) GetLoggingLevel() string {
	return c.runtimeConfig.Logging.Level
}

func (c *Config) GetLoggingFormat() string {
	return c.runtimeConfig.Logging.Format
}

func (c *Config) GetPluginDirs() []string {
	return c.runtimeConfig.Plugins.Dirs
}

func (c *Config) GetRuntimeConfig() *config {
	return c.runtimeConfig
}

// TODO: add a test
func (c *Config) GetCurrentWorkspace() string {
	return c.runtimeConfig.CurrentWorkspace
}

func (c *Config) loadFromFile() (config, error) {
	contents, err := afero.ReadFile(c.fs, c.persistedPath)

	if err != nil {
		return config{}, err
	}

	cfg := buildDefaultConfig()
	err = yaml.Unmarshal(contents, cfg)

	return *cfg, err
}

func (c *Config) LoadOrCreate() error {
	exists, err := afero.Exists(c.fs, c.persistedPath)

	if err != nil {
		return err
	}

	if !exists {
		c.persisted = buildDefaultConfig()
		c.runtimeConfig = buildDefaultConfig()

		return c.save()
	}

	cfg, err := c.loadFromFile()
	// Ensure we don't end up pointing at the same thing, to avoid confusion when
	// writing the config, that may override temporary changes applied by CLI args
	// or env vars
	persisted := cfg

	c.runtimeConfig = &persisted
	c.persisted = &cfg

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
