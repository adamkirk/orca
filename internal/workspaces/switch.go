package workspaces

import (
	"context"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
)

type SwitchCommand struct {
	bus.Returns[bus.Void]

	Workspace string
}

func (SwitchCommand) MessageName() string {
	return "ws.switch"
}

func (m *Manager) Switch(_ context.Context, cmd SwitchCommand) error {
	if err := m.configManager.SwitchWorkspace(cmd.Workspace); err != nil {
		return m.tui.RecordIfError("Failed to switch workspace!", err)
	}

	m.tui.Success(fmt.Sprintf("Switched to %s", cmd.Workspace))

	return nil
}
