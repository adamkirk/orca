package extension

import (
	"context"
	"fmt"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

// HookChoicesQuery lists the hooks that run something in the context: the
// project's, and the workspace's init.post. Each is described by the
// extensions it runs.
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

	if p := execCtx.Project; p != nil {
		for _, hook := range p.DefinedHooks() {
			names := []string{}

			for _, e := range p.Config.Hooks[hook] {
				names = append(names, e.Extension)
			}

			choices = append(choices, hookChoice(hook, names))
		}
	}

	if init := execCtx.Workspace.InitHooks; len(init) > 0 {
		names := make([]string, len(init))

		for i, e := range init {
			names[i] = fmt.Sprintf("%s's %s", e.Project, e.Extension)
		}

		choices = append(choices, hookChoice(workspace.HookInitPost, names))
	}

	return choices, nil
}

func hookChoice(hook workspace.Hook, names []string) workspace.Choice {
	return workspace.Choice{Value: hook.String(), Description: "Runs " + strings.Join(names, ", ")}
}
