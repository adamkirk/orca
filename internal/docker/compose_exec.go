package docker

import (
	"errors"
	"io"
	"os/exec"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// Exec runs a command in the service, in its running container.
// With output, the command isn't attached to a terminal, and what it writes goes
// to output, e.g. when a hook runs it. Without, it uses the host's terminal, so
// it can be interactive.
//
// TODO: guard against nil inputs
func (c *Compose) Exec(ws *workspace.Workspace, p *workspace.Project, service string, cmdArgs []string, output io.Writer) error {
	if err := c.goToProject(p); err != nil {
		return err
	}

	overlay, err := c.getOverlay(ws, p)
	if err != nil {
		return usererr.Wrap(err, "Failed to generate overlays!")
	}

	cmd := buildBaseComposeCommand(ws, p, overlay)
	cmd = append(cmd, attachArgs("exec", output)...)
	cmd = append(cmd, service)
	cmd = append(cmd, cmdArgs...)

	err = c.cli.Exec(cmd[0], cmd[1:], outputOpt(output), process.ChdirOpt(p.ProjectDir))

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
