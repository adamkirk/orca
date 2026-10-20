package extension

import (
	"context"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

// HookChoicesQuery lists the hooks that run something in the context: the
// project's, plus the workspace's that only it has, or outside of a project,
// the workspace's. Each is described by the extensions it runs.
type HookChoicesQuery struct {
	bus.Returns[[]workspace.Choice]

	Workspace string
	Project   string
}

func (HookChoicesQuery) MessageName() string {
	return "choices.hooks"
}

func (m *Manager) HookChoices(ctx context.Context, _ HookChoicesQuery) ([]workspace.Choice, error) {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return nil, err
	}

	choices := []workspace.Choice{}
	ws := execCtx.Workspace

	if execCtx.Project != nil {
		for _, hook := range execCtx.Project.DefinedHooks() {
			choices = append(choices, hookChoice(hook, execCtx.Project.Config.Hooks[hook]))
		}
	}

	for _, hook := range ws.DefinedHooks() {
		if hookProject(execCtx, hook) == nil {
			choices = append(choices, hookChoice(hook, ws.Hooks[hook]))
		}
	}

	return choices, nil
}

func hookChoice(hook workspace.Hook, entries []workspace.HookEntry) workspace.Choice {
	names := make([]string, len(entries))

	for i, e := range entries {
		names[i] = e.Extension
	}

	return workspace.Choice{Value: hook.String(), Description: "Runs " + strings.Join(names, ", ")}
}
