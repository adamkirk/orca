package docker

import (
	"github.com/adamkirk/orca/internal/workspace"
)

// TODO: guard against nil arguments
func (c *Compose) DoesSvcExist(ws *workspace.Workspace, p *workspace.Project, service string) (bool, error) {
	spec, err := c.LoadSpec(ws, p)

	if err != nil {
		c.tui.RecordIfError("Failed to load docker compose spec for project", err)
		return false, err
	}

	_, ok := spec.Services[service]

	return ok, nil
}
