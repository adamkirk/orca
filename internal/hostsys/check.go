package hostsys

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

func (hs *HostSystem) CheckSetup(_ context.Context, query CheckSetupQuery) error {
	if err := hs.VerifySetup(); err != nil {
		return err
	}

	hs.tui.Success("All requirements met!")

	return nil
}
