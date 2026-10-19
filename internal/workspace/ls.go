package workspace

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
)

type LsQuery struct {
	bus.Returns[bus.Void]
}

func (LsQuery) MessageName() string {
	return "ws.ls"
}

func (m *Manager) Ls(_ context.Context, query LsQuery) error {
	current := m.configManager.GetCurrentWorkspace()
	locs := m.configManager.GetAllWorkspaceMeta()

	if len(locs) == 0 {
		m.tui.Error("No workspaces exist!")

		return nil
	}

	rows := make([][]string, len(locs))

	for i, loc := range locs {
		inUse := "No"

		if loc.Name == current {
			inUse = "Yes"
		}
		rows[i] = []string{
			inUse,
			loc.Name,
			loc.Path,
		}
	}

	m.tui.Table([]string{
		"In Use?",
		"Project",
		"Path",
	}, rows)

	return nil
}
