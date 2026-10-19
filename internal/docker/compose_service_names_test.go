package docker

import (
	"errors"
	"testing"

	"github.com/adamkirk/orca/internal/workspace"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubParser returns the project, or error, it's given, and records what it
// was asked to parse.
type stubParser struct {
	project *types.Project
	err     error

	paths    []string
	envFiles []string
}

func (p *stubParser) Parse(paths []string, envFiles []string) (*types.Project, error) {
	p.paths = paths
	p.envFiles = envFiles

	return p.project, p.err
}

func Test_ServiceNames(t *testing.T) {
	project := &workspace.Project{
		ProjectDir: "/code/api",
		Config: workspace.ProjectConfig{
			ComposeFiles: workspace.ComposeFiles{Primary: "docker-compose.yml"},
			EnvFiles:     []workspace.EnvFile{{Path: ".env"}, {Path: "/etc/orca/shared.env"}},
		},
	}

	t.Run("sorted, from the project's compose file", func(t *testing.T) {
		parser := &stubParser{project: &types.Project{Services: types.Services{
			"worker": {Name: "worker"},
			"app":    {Name: "app"},
			"db":     {Name: "db"},
		}}}

		c := NewCompose(nil, nil, nil, parser)

		names, err := c.ServiceNames(project)

		require.NoError(t, err)
		assert.Equal(t, []string{"app", "db", "worker"}, names)
		assert.Equal(t, []string{"/code/api/docker-compose.yml"}, parser.paths)
		assert.Equal(t, []string{"/code/api/.env", "/etc/orca/shared.env"}, parser.envFiles, "relative to the project")
	})

	t.Run("a compose file that can't be parsed", func(t *testing.T) {
		parseErr := errors.New("invalid compose file")
		c := NewCompose(nil, nil, nil, &stubParser{err: parseErr})

		_, err := c.ServiceNames(project)

		assert.ErrorIs(t, err, parseErr)
	})
}
