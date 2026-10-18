package docker

import (
	"strings"

	"github.com/adamkirk/orca/internal/workspace"
)

// TODO: guard against nil inputs
func (c *Compose) ShowCommand(ws *workspace.Workspace, p *workspace.Project) error {
	if err := c.goToProject(p); err != nil {
		return err
	}

	overlay, err := c.getOverlay(ws, p)
	if err != nil {
		return c.tui.RecordIfError("Failed to generate overlays!", err)
	}

	cmd := buildBaseComposeCommand(ws, p, overlay)

	c.tui.Info(strings.Join(cmd, " "))

	return nil
}
