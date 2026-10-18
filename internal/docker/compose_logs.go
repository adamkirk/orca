package docker

import (
	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// TODO: guard against nil arguments
func (c *Compose) Logs(ws *workspace.Workspace, p *workspace.Project, service string) error {
	if err := c.goToProject(p); err != nil {
		return err
	}

	overlay, err := c.getOverlay(ws, p)
	if err != nil {
		return usererr.Wrap(err, "Failed to generate overlays!")
	}

	cmd := buildBaseComposeCommand(ws, p, overlay)
	cmd = append(cmd, "logs", "-f")
	if service != "" {
		cmd = append(cmd, service)
	}

	err = c.cli.Exec(cmd[0], cmd[1:], process.WithHostIO(), process.ChdirOpt(p.ProjectDir))

	if err != nil {
		return usererr.Wrap(err, "Failed to tail logs!")
	}

	return nil
}
