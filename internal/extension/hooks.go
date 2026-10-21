package extension

import (
	"fmt"

	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// planHook plans the extensions one of the project's hooks runs, in order.
// They're all the project's own.
func planHook(ws *workspace.Workspace, project *workspace.Project, hook workspace.Hook) ([]planned, error) {
	t := target{ws: ws, project: project}
	entries := project.Config.Hooks[hook]
	runs := make([]planned, len(entries))

	for i, entry := range entries {
		ext, err := project.FindExtension(entry.Extension)

		// Loading the config checked they exist, so this only fails on a bug
		if err != nil {
			return nil, fmt.Errorf("planning %s's %s: %w", project.Name, hook, err)
		}

		runs[i] = planned{hook: hook.String(), target: t, ext: ext}
	}

	return runs, nil
}

// planInit plans the workspace's init.post. Each entry is one of a project's
// extensions, which runs in that project.
func planInit(ws *workspace.Workspace) ([]planned, error) {
	runs := make([]planned, len(ws.InitHooks))

	for i, entry := range ws.InitHooks {
		project, err := ws.GetProject(entry.Project)

		if err != nil {
			return nil, fmt.Errorf("planning init.post: %w", err)
		}

		// Its extensions aren't known until it's cloned
		if !project.IsRegistered {
			return nil, usererr.Wrap(
				fmt.Errorf("%w: %s", ErrProjectNotCloned, project.Name),
				fmt.Sprintf("init.post runs %s's '%s', but %s isn't cloned; clone it with 'orca ws clone -p %s'.", project.Name, entry.Extension, project.Name, project.Name),
			)
		}

		ext, err := project.FindExtension(entry.Extension)

		if err != nil {
			return nil, fmt.Errorf("planning init.post: %w", err)
		}

		runs[i] = planned{hook: workspace.HookInitPost.String(), target: target{ws: ws, project: project}, ext: ext}
	}

	return runs, nil
}

// runPlanned runs each extension in turn, stopping at the first that fails.
// Each is shown with the hook running it, and its output under it.
func (m *Manager) runPlanned(runs []planned) error {
	for _, run := range runs {
		project := run.target.project.Name

		m.tui.Info(fmt.Sprintf("%s %s: %s", project, run.hook, run.ext.Name))

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

// planHooks plans the extensions of each of the hooks, in order.
func planHooks(ws *workspace.Workspace, runs []workspace.HookRun) ([]planned, error) {
	all := []planned{}

	for _, run := range runs {
		planned, err := planHook(ws, run.Project, run.Hook)

		if err != nil {
			return nil, err
		}

		all = append(all, planned...)
	}

	return all, nil
}

// ConfirmHooks asks about every extension the hooks will run, once, before any
// of them do, so they can be run, with RunConfirmedHook, in between other work.
func (m *Manager) ConfirmHooks(ws *workspace.Workspace, runs []workspace.HookRun, yes bool) error {
	planned, err := planHooks(ws, runs)

	if err != nil {
		return err
	}

	return m.confirm(planned, yes)
}

// RunConfirmedHook runs one of the hooks ConfirmHooks has already asked about.
func (m *Manager) RunConfirmedHook(ws *workspace.Workspace, run workspace.HookRun) error {
	planned, err := planHook(ws, run.Project, run.Hook)

	if err != nil {
		return err
	}

	return m.runPlanned(planned)
}

// SetUp runs the clone.post hook of each of the projects, then the workspace's
// init.post, if asked for, which runs projects' extensions. Everything they'll run is asked about first, in one
// go, and it all stops at the first extension that fails.
func (m *Manager) SetUp(ws *workspace.Workspace, opts workspace.SetUpOpts) error {
	runs := []workspace.HookRun{}

	for _, name := range opts.Projects {
		project, err := ws.GetProject(name)

		if err != nil {
			return err
		}

		runs = append(runs, workspace.HookRun{Project: project, Hook: workspace.HookClonePost})
	}

	planned, err := planHooks(ws, runs)

	if err != nil {
		return err
	}

	if opts.Initialise {
		initRuns, err := planInit(ws)

		if err != nil {
			return err
		}

		planned = append(planned, initRuns...)
	}

	if len(planned) == 0 {
		return nil
	}

	if err := m.confirm(planned, opts.Yes); err != nil {
		return err
	}

	return m.runPlanned(planned)
}
