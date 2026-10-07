package common

import (
	"os"
	"path/filepath"
	"strings"
)

type ExecutionContext struct {
	Workspace        *Workspace
	Project          *Project
	WorkingDirectory string
}

type configManager interface {
	GetAllProjectMeta() []ProjectMeta
	GetCurrentWorkspace() string
	GetWorkspaceMeta(name string) (WorkspaceMeta, error)
}

type workspaceRepository interface {
	Load(name string) (*Workspace, error)
}

type ContextResolver struct {
	cfg           configManager
	workspaceRepo workspaceRepository
}

func (c *ContextResolver) getProjectFromWorkdir() (*ProjectMeta, error) {
	dir, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	return findProjectForDir(dir, c.cfg.GetAllProjectMeta()), nil
}

// findProjectForDir returns the project whose directory contains dir, or nil if
// there isn't one. Projects can be nested inside each other, so the most specific
// (deepest) match wins, regardless of the order the projects are in.
func findProjectForDir(dir string, projects []ProjectMeta) *ProjectMeta {
	dir = filepath.Clean(dir)

	var found *ProjectMeta
	foundDepth := -1

	for i := range projects {
		path := filepath.Clean(projects[i].Path)

		if !isWithinDir(dir, path) {
			continue
		}

		if depth := strings.Count(path, string(filepath.Separator)); depth > foundDepth {
			found = &projects[i]
			foundDepth = depth
		}
	}

	return found
}

// isWithinDir reports whether dir is parent, or anywhere below it. Both must be
// cleaned paths.
func isWithinDir(dir string, parent string) bool {
	if dir == parent {
		return true
	}

	// Compare with a trailing separator, so /code/api doesn't match /code/api-v2
	prefix := strings.TrimSuffix(parent, string(filepath.Separator)) + string(filepath.Separator)

	return strings.HasPrefix(dir, prefix)
}

func (c *ContextResolver) buildExecutionContext(wsName string, projectName string) (ExecutionContext, error) {
	workDir, err := os.Getwd()

	if err != nil {
		return ExecutionContext{}, err
	}

	meta, err := c.cfg.GetWorkspaceMeta(wsName)

	if err != nil {
		return ExecutionContext{}, err
	}

	ws, err := c.workspaceRepo.Load(meta.Name)

	if err != nil {
		return ExecutionContext{}, err
	}

	if projectName == "" {
		return ExecutionContext{
			Workspace:        ws,
			WorkingDirectory: workDir,
		}, nil
	}

	project, err := ws.GetProject(projectName)

	if err != nil {
		return ExecutionContext{}, err
	}

	return ExecutionContext{
		Workspace:        ws,
		Project:          project,
		WorkingDirectory: workDir,
	}, nil
}

func (c *ContextResolver) Resolve(ws string, project string) (ExecutionContext, error) {
	// We've got a specific workspace and project
	if ws != "" && project != "" {
		return c.buildExecutionContext(ws, project)
	}

	// No workspace, but a project was specified, so we'll assume that it's the
	// current workspace
	if ws == "" && project != "" {
		p, err := c.getProjectFromWorkdir()

		if err != nil {
			return ExecutionContext{}, err
		}

		if p == nil {
			ws = c.cfg.GetCurrentWorkspace()
		} else {
			ws = p.WorkspaceName
		}

		// If the current workspace is not defined
		if ws == "" {
			return ExecutionContext{}, ErrCouldNotDetermineWorkspace{
				Message: "no global workspace chosen, and not in an orca project directory",
			}
		}

		return c.buildExecutionContext(ws, project)
	}

	// The workspace was specified, but not project was so we'll use that workspace
	// and assume all projects.
	if ws != "" && project == "" {
		return c.buildExecutionContext(ws, "")
	}

	// If we get here, both of the options were empty, so we'll try resolve from
	// the working directory
	p, err := c.getProjectFromWorkdir()

	if err != nil {
		return ExecutionContext{}, err
	}

	if p != nil {
		return c.buildExecutionContext(p.WorkspaceName, p.Name)
	}

	// We weren't in a project directory, so our final resort is to use the
	// current workspace, and assume all projects should be started.
	current := c.cfg.GetCurrentWorkspace()

	if current == "" {
		return ExecutionContext{}, ErrCouldNotDetermineWorkspace{
			Message: "no global workspace chosen, and not in an orca project directory",
		}
	}

	return c.buildExecutionContext(current, "")
}

func NewContextResolver(cfg configManager, workspaceRepo workspaceRepository) *ContextResolver {
	return &ContextResolver{
		cfg:           cfg,
		workspaceRepo: workspaceRepo,
	}
}
