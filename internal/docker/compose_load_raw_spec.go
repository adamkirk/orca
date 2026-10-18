package docker

import (
	"errors"

	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/process"
)

// TODO: guard against nil arguments
func (c *Compose) LoadRawSpec(ws *common.Workspace, p *common.Project) (string, error) {
	if err := c.goToProject(p); err != nil {
		return "", err
	}

	overlay, err := c.getOverlay(ws, p)
	if err != nil {
		return "", c.tui.RecordIfError("Failed to generate overlays!", err)
	}

	cmd := buildBaseComposeCommand(ws, p, overlay)
	cmd = append(cmd, "config")

	stdErrOpt, stderr := process.WithStderr()
	stdOutOpt, stdout := process.WithStdout()

	err = c.cli.Exec(cmd[0], cmd[1:], stdErrOpt, stdOutOpt, process.ChdirOpt(p.ProjectDir))

	if err != nil {
		return "", c.tui.RecordIfError("Failed to display config!", errors.New(stderr.String()))
	}

	return stdout.String(), nil
}
