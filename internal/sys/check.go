package sys

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
)

type CheckSetupQuery struct {
	bus.Returns[bus.Void]
}

func (CheckSetupQuery) MessageName() string {
	return "sys.check"
}

func (m *Manager) CheckSetup(_ context.Context, query CheckSetupQuery) error {
	if err := m.VerifySetup(); err != nil {
		return err
	}

	m.tui.Success("All requirements met!")

	return nil
}
