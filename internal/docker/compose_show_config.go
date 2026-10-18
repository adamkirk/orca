package docker

import (
	"github.com/adamkirk/orca/internal/workspace"
)

// TODO: guard against nil inputs
func (c *Compose) ShowConfig(ws *workspace.Workspace, p *workspace.Project) error {
	spec, err := c.LoadRawSpec(ws, p)

	if err != nil {
		return err
	}

	c.tui.Info(spec)

	return nil
}
