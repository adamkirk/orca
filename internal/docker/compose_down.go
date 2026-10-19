package docker

import (
	"fmt"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// TODO: guard against nil arguments
func (c *Compose) Down(ws *workspace.Workspace, p *workspace.Project) error {
	if err := c.goToProject(p); err != nil {
		return err
	}

	overlay, err := c.getOverlay(ws, p)
	if err != nil {
		return usererr.Wrap(err, "Failed to generate overlays!")
	}

	c.tui.Info(fmt.Sprintf("%s:%s[%s] stopping...", p.Name, ws.Name, p.ProjectDir))
	c.tui.NewLine()
	cmd := buildBaseComposeCommand(ws, p, overlay)
	cmd = append(cmd, "down", "--remove-orphans")

	err = c.cli.Exec(cmd[0], cmd[1:], process.WithHostIO(), process.ChdirOpt(p.ProjectDir))

	if err != nil {
		return usererr.Wrap(err, fmt.Sprintf("%s:%s[%s] failed!", p.Name, ws.Name, p.ProjectDir))
	}

	c.tui.Success(fmt.Sprintf("%s:%s[%s] stopped!", p.Name, ws.Name, p.ProjectDir))
	c.tui.NewLine()

	return nil
}
