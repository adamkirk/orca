package hosts

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

type tui interface {
	Success(msg ...string)
	RecordIfError(msg string, err error) error
}

type etcHostsManager interface {
	SyncForWorkspace(ws *workspace.Workspace) error
}

// Manager handles the hosts command, which adds a workspace's hosts to the
// /etc/hosts file.
type Manager struct {
	etcHosts etcHostsManager
	tui      tui
}

func NewManager(etcHosts etcHostsManager, tui tui) *Manager {
	return &Manager{
		etcHosts: etcHosts,
		tui:      tui,
	}
}

type SyncCommand struct {
	bus.Returns[bus.Void]

	Workspace string
}

func (SyncCommand) MessageName() string {
	return "hosts"
}

func (m *Manager) Sync(ctx context.Context, cmd SyncCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	if err := m.etcHosts.SyncForWorkspace(execCtx.Workspace); err != nil {
		return m.tui.RecordIfError("Failed to save /etc/hosts file!", err)
	}

	m.tui.Success("Hosts are now added to the /etc/hosts file!")

	return nil
}
