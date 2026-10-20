package extension

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// RunHookCommand runs a hook by hand: the project's, or the workspace's for a
// hook only the workspace has (init.post), or outside of any project.
type RunHookCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
	Hook      string

	// Run every extension without asking, and without recording them as trusted
	Yes bool
}

func (RunHookCommand) MessageName() string {
	return "hooks.run"
}

func (m *Manager) RunHook(ctx context.Context, cmd RunHookCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	hook, err := workspace.ParseHook(cmd.Hook)

	if err != nil {
		return usererr.Wrap(err, fmt.Sprintf("There's no hook '%s'; a project can have %s, and the workspace %s.", cmd.Hook, hookList(workspace.ProjectHooks), hookList(workspace.WorkspaceHooks)))
	}

	project := hookProject(execCtx, hook)

	if project == nil && !slices.Contains(workspace.WorkspaceHooks, hook) {
		return usererr.Wrap(
			fmt.Errorf("%w: %s needs a project", workspace.ErrUnknownHook, hook),
			fmt.Sprintf("%s is a project's hook; run it from inside the project, or pass it with -p.", hook),
		)
	}

	runs, err := planHook(execCtx.Workspace, project, hook)

	if err != nil {
		return err
	}

	if len(runs) == 0 {
		where := "The workspace"

		if project != nil {
			where = project.Name
		}

		m.tui.Info(fmt.Sprintf("%s has no %s hook, so there's nothing to run.", where, hook))

		return nil
	}

	if err := m.confirm(runs, cmd.Yes); err != nil {
		return err
	}

	return m.runPlanned(runs)
}

// hookProject is the project whose hook runs: the one in the context, unless
// only the workspace has the hook.
func hookProject(execCtx workspace.ExecutionContext, hook workspace.Hook) *workspace.Project {
	if execCtx.Project == nil || !slices.Contains(workspace.ProjectHooks, hook) {
		return nil
	}

	return execCtx.Project
}

func hookList(hooks []workspace.Hook) string {
	names := make([]string, len(hooks))

	for i, h := range hooks {
		names[i] = h.String()
	}

	return strings.Join(names, ", ")
}
