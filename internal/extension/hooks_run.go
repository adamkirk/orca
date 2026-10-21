package extension

import (
	"context"
	"fmt"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// RunHookCommand runs a hook by hand: one of the project's, or the
// workspace's init.post, from anywhere.
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
		return usererr.Wrap(err, fmt.Sprintf("There's no hook '%s'; a project can have %s, and the workspace init.post.", cmd.Hook, hookList(workspace.ProjectHooks)))
	}

	runs, where, err := planHookRun(execCtx, hook)

	if err != nil {
		return err
	}

	if len(runs) == 0 {
		m.tui.Info(fmt.Sprintf("%s has no %s hook, so there's nothing to run.", where, hook))

		return nil
	}

	if err := m.confirm(runs, cmd.Yes); err != nil {
		return err
	}

	return m.runPlanned(runs)
}

// planHookRun plans the hook asked for: the workspace's init.post, from
// anywhere, or the project's hook. It also says whose hook it is, for messages.
func planHookRun(execCtx workspace.ExecutionContext, hook workspace.Hook) ([]planned, string, error) {
	if hook == workspace.HookInitPost {
		runs, err := planInit(execCtx.Workspace)

		return runs, "The workspace", err
	}

	if execCtx.Project == nil {
		return nil, "", usererr.Wrap(
			fmt.Errorf("%w: %s needs a project", workspace.ErrUnknownHook, hook),
			fmt.Sprintf("%s is a project's hook; run it from inside the project, or pass it with -p.", hook),
		)
	}

	runs, err := planHook(execCtx.Workspace, execCtx.Project, hook)

	return runs, execCtx.Project.Name, err
}

func hookList(hooks []workspace.Hook) string {
	names := make([]string, len(hooks))

	for i, h := range hooks {
		names[i] = h.String()
	}

	return strings.Join(names, ", ")
}
