package docker

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/workspace"
)

// IsProjectRunning reports whether any of the project's containers are
// running. It finds them by their compose project label, so it works without
// the project's compose files, even for a project that isn't cloned.
func (c *Compose) IsProjectRunning(ws *workspace.Workspace, p *workspace.Project) (bool, error) {
	withStdout, outBuff := process.WithStdout()
	withStderr, errBuff := process.WithStderr()

	err := c.cli.Exec("docker", []string{
		"ps",
		"-q",
		"--filter",
		fmt.Sprintf("label=com.docker.compose.project=%s", composeProjectName(ws, p)),
	}, withStdout, withStderr)

	if err != nil {
		slog.Debug("stderr from docker ps", "stderr", errBuff.String())

		return false, fmt.Errorf("checking whether %s is running: %w", p.Name, err)
	}

	return strings.TrimSpace(outBuff.String()) != "", nil
}
