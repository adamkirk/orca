package docker

import (
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/compose-spec/compose-go/v2/types"
	"go.yaml.in/yaml/v4"
)

// TODO: guard against nil arguments
func (c *Compose) LoadSpec(ws *workspace.Workspace, p *workspace.Project) (*types.Project, error) {
	rawSpec, err := c.LoadRawSpec(ws, p)

	if err != nil {
		return nil, err
	}

	spec := &types.Project{}

	err = yaml.Unmarshal([]byte(rawSpec), spec)

	return spec, err
}
