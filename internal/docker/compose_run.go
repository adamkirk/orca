package docker

import (
	"errors"
	"os/exec"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// TODO: guard against nil inputs
func (c *Compose) Run(ws *workspace.Workspace, p *workspace.Project, service string, cmdArgs []string) error {
	if err := c.goToProject(p); err != nil {
		return err
	}

	overlay, err := c.getOverlay(ws, p)
	if err != nil {
		return usererr.Wrap(err, "Failed to generate overlays!")
	}

	cmd := buildBaseComposeCommand(ws, p, overlay)
	cmd = append(cmd, "run", "-it", "--rm", service)
	cmd = append(cmd, cmdArgs...)

	err = c.cli.Exec(cmd[0], cmd[1:], process.WithHostIO(), process.ChdirOpt(p.ProjectDir))

	if err != nil {
		// If it's exit code 130, it's because they exited an interactive container
		// typically. Not sure if this is always true though, need to see how this
		// works across more use-cases.
		if exitError, ok := errors.AsType[*exec.ExitError](err); ok && exitError.ProcessState.ExitCode() == 130 {
			return nil
		}

		return usererr.Wrap(err, "Command failed!")
	}

	return nil
}
