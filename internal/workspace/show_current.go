package workspace

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
)

type ShowCurrentQuery struct {
	bus.Returns[bus.Void]
}

func (ShowCurrentQuery) MessageName() string {
	return "ws.current"
}

func (m *Manager) ShowCurrent(_ context.Context, query ShowCurrentQuery) error {
	current := m.configManager.GetCurrentWorkspace()

	if current == "" {
		m.tui.Info("<no workspace selected>")
		return nil
	}

	m.tui.Info(current)
	return nil
}
