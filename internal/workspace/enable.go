package workspace

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
)

type EnableCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Projects  []string
}

func (EnableCommand) MessageName() string {
	return "ws.enable"
}

// checkProjectNames returns the names without duplicates, failing if any
// aren't in the workspace.
func checkProjectNames(ws *Workspace, names []string) ([]string, error) {
	checked := []string{}
	unknown := []string{}

	for _, name := range names {
		if slices.Contains(checked, name) || slices.Contains(unknown, name) {
			continue
		}

		if _, err := ws.GetProject(name); err != nil {
			unknown = append(unknown, name)
			continue
		}

		checked = append(checked, name)
	}

	if len(unknown) > 0 {
		return nil, usererr.Wrap(
			fmt.Errorf("%w: %s", ErrUnknownProject, strings.Join(unknown, ", ")),
			fmt.Sprintf("Workspace '%s' has no project %s!", ws.Name, joinNames(unknown, "or")),
		)
	}

	if len(checked) == 0 {
		return nil, usererr.Wrap(
			fmt.Errorf("%w: no projects given", ErrInvalidInput),
			"Must supply at least one project!",
		)
	}

	return checked, nil
}

// joinNames lists names in a sentence, e.g. "web, admin and api".
func joinNames(names []string, conjunction string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}

	return fmt.Sprintf("%s %s %s", strings.Join(names[:len(names)-1], ", "), conjunction, names[len(names)-1])
}

func (m *Manager) Enable(ctx context.Context, cmd EnableCommand) error {
	execCtx, err := ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	names, err := checkProjectNames(execCtx.Workspace, cmd.Projects)

	if err != nil {
		return err
	}

	if err := setProjectsEnabled(m.fs, execCtx.Workspace.PreferencesPath, names, true); err != nil {
		return usererr.Wrap(err, "Failed to update the preferences file!")
	}

	m.tui.Success(fmt.Sprintf("Enabled %s.", joinNames(names, "and")))

	return nil
}
