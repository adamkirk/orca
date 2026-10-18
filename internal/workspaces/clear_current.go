package workspaces

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
)

type ClearCurrentCommand struct {
	bus.Returns[bus.Void]
}

func (ClearCurrentCommand) MessageName() string {
	return "ws.clear-current"
}

func (m *Manager) ClearCurrent(_ context.Context, cmd ClearCurrentCommand) error {
	current := m.configManager.GetCurrentWorkspace()

	if current == "" {
		m.tui.Info("<no workspace selected>")
		return nil
	}

	return m.configManager.ClearCurrentWorkspace()
}
