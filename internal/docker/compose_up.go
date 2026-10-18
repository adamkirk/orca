package docker

import (
	"fmt"

	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/process"
)

// TODO: guard against nil arguments
func (c *Compose) Up(ws *common.Workspace, p *common.Project) error {
	if err := c.goToProject(p); err != nil {
		return err
	}

	overlay, err := c.getOverlay(ws, p)
	if err != nil {
		return c.tui.RecordIfError("Failed to generate overlays!", err)
	}

	c.tui.Info(fmt.Sprintf("%s:%s[%s] starting...", p.Name, ws.Name, p.ProjectDir))
	c.tui.NewLine()
	cmd := buildBaseComposeCommand(ws, p, overlay)
	cmd = append(cmd, "up", "-d")

	err = c.cli.Exec(cmd[0], cmd[1:], process.WithHostIO(), process.ChdirOpt(p.ProjectDir))

	if err != nil {
		return c.tui.RecordIfError(fmt.Sprintf("%s:%s[%s] failed!", p.Name, ws.Name, p.ProjectDir), err)
	}

	c.tui.Success(fmt.Sprintf("%s:%s[%s] complete!", p.Name, ws.Name, p.ProjectDir))
	c.tui.NewLine()

	return nil
}
