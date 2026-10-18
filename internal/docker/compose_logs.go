package docker

import (
	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/process"
)

// TODO: guard against nil arguments
func (c *Compose) Logs(ws *common.Workspace, p *common.Project, service string) error {
	if err := c.goToProject(p); err != nil {
		return err
	}

	overlay, err := c.getOverlay(ws, p)
	if err != nil {
		return c.tui.RecordIfError("Failed to generate overlays!", err)
	}

	cmd := buildBaseComposeCommand(ws, p, overlay)
	cmd = append(cmd, "logs", "-f")
	if service != "" {
		cmd = append(cmd, service)
	}

	err = c.cli.Exec(cmd[0], cmd[1:], process.WithHostIO(), process.ChdirOpt(p.ProjectDir))

	if err != nil {
		return c.tui.RecordIfError("Failed to tail logs!", err)
	}

	return nil
}
