package extension

import (
	"fmt"

	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// planHook plans the extensions a hook runs, in order: a project's hook, or the
// workspace's when project is nil. Every extension runs in the project, even
// one defined in the workspace config.
func planHook(ws *workspace.Workspace, project *workspace.Project, hook workspace.Hook) ([]planned, error) {
	t := target{ws: ws, project: project}
	entries := ws.Hooks[hook]

	if project != nil {
		entries = project.Config.Hooks[hook]
	}

	runs := make([]planned, len(entries))

	for i, entry := range entries {
		run := planned{hook: hook.String(), target: t}

		var err error

		// Loading the config checked these exist, so this only fails on a bug
		switch entry.Scope {
		case workspace.HookScopeProject:
			run.owner = project
			run.ext, err = project.FindExtension(entry.Extension)
		case workspace.HookScopeWorkspace:
			run.ext, err = ws.FindExtension(entry.Extension)
		default:
			err = fmt.Errorf("unknown hook scope '%s'", entry.Scope)
		}

		if err != nil {
			return nil, fmt.Errorf("planning %s: %w", hook, err)
		}

		runs[i] = run
	}

	return runs, nil
}

// runPlanned runs each extension in turn, stopping at the first that fails.
// Each is shown with the hook running it, and its output under it.
func (m *Manager) runPlanned(runs []planned) error {
	for _, run := range runs {
		project := ""
		label := fmt.Sprintf("%s: %s", run.hook, run.ext.Name)

		if run.target.project != nil {
			project = run.target.project.Name
			label = fmt.Sprintf("%s %s", project, label)
		}

		m.tui.Info(label)

		if err := m.execute(run.target, run.ext, nil, "  "); err != nil {
			return HookFailedError{
				Workspace: run.target.ws.Name,
				Project:   project,
				Hook:      run.hook,
				Extension: run.ext.Name,

				// Only shown when the extension didn't say what went wrong
				Err: usererr.Wrap(err, fmt.Sprintf("Extension '%s' failed!", run.ext.Name)),
			}
		}
	}

	return nil
}

// SetUp runs the clone.post hook of each of the projects, then the workspace's
// init.post, if asked for. Everything they'll run is asked about first, in one
// go, and it all stops at the first extension that fails.
func (m *Manager) SetUp(ws *workspace.Workspace, opts workspace.SetUpOpts) error {
	runs := []planned{}

	for _, name := range opts.Projects {
		project, err := ws.GetProject(name)

		if err != nil {
			return err
		}

		projectRuns, err := planHook(ws, project, workspace.HookClonePost)

		if err != nil {
			return err
		}

		runs = append(runs, projectRuns...)
	}

	if opts.Initialise {
		wsRuns, err := planHook(ws, nil, workspace.HookInitPost)

		if err != nil {
			return err
		}

		runs = append(runs, wsRuns...)
	}

	if len(runs) == 0 {
		return nil
	}

	if err := m.confirm(runs, opts.Yes); err != nil {
		return err
	}

	return m.runPlanned(runs)
}
