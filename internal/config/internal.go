package config

import (
	"fmt"
	"slices"

	"github.com/adamkirk/orca/internal/workspace"
)

type configLogging struct {
	Level  string
	Format string
}

type configProject struct {
	Name string
	Path string
}

type configWorkspace struct {
	Name     string
	Path     string
	Projects []configProject
}

type configPlugins struct {
	// Dirs are searched for plugin executables, in addition to the default
	// plugins directory.
	Dirs []string `yaml:"dirs,omitempty"`
}

type config struct {
	Logging          configLogging
	Workspaces       []configWorkspace
	CurrentWorkspace string        `yaml:"currentWorkspace" mapstructure:"current_workspace"`
	Plugins          configPlugins `yaml:"plugins,omitempty"`
}

func (c *config) workspaceIndex(name string) int {
	return slices.IndexFunc(c.Workspaces, func(ws configWorkspace) bool {
		return ws.Name == name
	})
}

func (ws configWorkspace) projectIndex(name string) int {
	return slices.IndexFunc(ws.Projects, func(p configProject) bool {
		return p.Name == name
	})
}

// upsert returns a copy of items, with item replacing the element at i, or
// appended when i is -1. It's copied so the runtime and persisted configs never
// share the same backing array.
func upsert[T any](items []T, i int, item T) []T {
	items = slices.Clone(items)

	if i == -1 {
		return append(items, item)
	}

	items[i] = item

	return items
}

func (c *config) workspaceExists(name string) bool {
	return c.workspaceIndex(name) != -1
}

func (c *config) getWorkspace(name string) (configWorkspace, error) {
	i := c.workspaceIndex(name)

	if i == -1 {
		return configWorkspace{}, fmt.Errorf("%w: %s", workspace.ErrUnknownWorkspace, name)
	}

	return c.Workspaces[i], nil
}

func (c *config) projectExists(wsName string, name string) (bool, error) {
	ws, err := c.getWorkspace(wsName)

	if err != nil {
		return false, err
	}

	return ws.projectIndex(name) != -1, nil
}

func (c *config) getProject(wsName string, name string) (configProject, error) {
	ws, err := c.getWorkspace(wsName)

	if err != nil {
		return configProject{}, err
	}

	i := ws.projectIndex(name)

	if i == -1 {
		return configProject{}, fmt.Errorf("%w: %s", workspace.ErrUnknownProject, name)
	}

	return ws.Projects[i], nil
}
