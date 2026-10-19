package docker

import (
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// TODO: guard against nil arguments
func (c *Compose) DoesSvcExist(ws *workspace.Workspace, p *workspace.Project, service string) (bool, error) {
	spec, err := c.LoadSpec(ws, p)

	if err != nil {
		return false, usererr.Wrap(err, "Failed to load docker compose spec for project")
	}

	_, ok := spec.Services[service]

	return ok, nil
}
