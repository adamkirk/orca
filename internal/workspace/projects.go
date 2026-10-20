package workspace

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/spf13/afero"
)

type ProjectsQuery struct {
	bus.Returns[bus.Void]

	Workspace string
}

func (ProjectsQuery) MessageName() string {
	return "ws.projects"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}

	return "no"
}

func (m *Manager) isCloned(p Project) bool {
	if !p.IsRegistered {
		return false
	}

	exists, err := afero.DirExists(m.fs, p.ProjectDir)

	return err == nil && exists
}

// Projects lists the workspace's projects, and whether each is enabled, cloned
// and running. When docker can't be asked, running is shown as '?', rather
// than failing.
func (m *Manager) Projects(ctx context.Context, _ ProjectsQuery) error {
	execCtx, err := ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	ws := execCtx.Workspace
	dockerAvailable := true
	rows := make([][]string, len(ws.Projects))

	for i, p := range ws.Projects {
		running := "?"

		if dockerAvailable {
			isRunning, err := m.stack.IsProjectRunning(ws, &p)

			if err != nil {
				slog.Debug("couldn't check whether projects are running", "err", err)
				dockerAvailable = false
			} else {
				running = yesNo(isRunning)
			}
		}

		rows[i] = []string{p.Name, "service", yesNo(!p.Disabled), yesNo(m.isCloned(p)), running}
	}

	m.tui.Table([]string{"Project", "Type", "Enabled", "Cloned", "Running"}, rows)

	for _, name := range ws.UnknownPreferenceProjects {
		m.tui.Info(fmt.Sprintf("Warning: %s lists '%s', which isn't in the workspace, so it's ignored.", ws.PreferencesPath, name))
	}

	return nil
}
