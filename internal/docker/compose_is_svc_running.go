package docker

import (
	"log/slog"
	"strings"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// TODO: guard against nil arguments
func (c *Compose) IsSvcRunning(ws *workspace.Workspace, p *workspace.Project, service string) (bool, error) {
	if err := c.goToProject(p); err != nil {
		return false, err
	}

	overlay, err := c.getOverlay(ws, p)
	if err != nil {
		return false, usererr.Wrap(err, "Failed to generate overlays!")
	}

	baseCmd := buildBaseComposeCommand(ws, p, overlay)
	psCmd := append(baseCmd, "ps", "-q", service)

	withStdout, outBuff := process.WithStdout()
	withStderr, errBuff := process.WithStderr()

	err = c.cli.Exec(psCmd[0], psCmd[1:], withStdout, withStderr, process.ChdirOpt(p.ProjectDir))

	if err != nil {
		slog.Debug("stderr from docker compose ps", "stderr", errBuff.String())
		return false, usererr.Wrap(err, "Failed to check if pod is running")
	}

	containerID := strings.TrimSpace(outBuff.String())
	if containerID == "" {
		// Service not found or not running at all
		return false, nil
	}

	withStdout, outBuff = process.WithStdout()
	withStderr, errBuff = process.WithStderr()

	err = c.cli.Exec("docker", []string{"inspect", "-f", "{{.State.Running}}", containerID}, withStdout, withStderr)

	if err != nil {
		slog.Debug("stderr from docker inspect", "stderr", errBuff.String())
		return false, usererr.Wrap(err, "Failed to inspect pod")
	}

	state := strings.TrimSpace(outBuff.String())

	return state == "true", nil
}
