package workspace

import (
	"context"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
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
		return usererr.Wrap(err, "Failed to switch workspace!")
	}

	m.tui.Success(fmt.Sprintf("Switched to %s", cmd.Workspace))

	return nil
}
