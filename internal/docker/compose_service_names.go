package docker

import (
	"path/filepath"

	"github.com/adamkirk/orca/internal/workspace"
)

// ServiceNames returns the names of the services in the project's compose
// file, sorted. The file is parsed directly, rather than asking docker, so it's
// quick, and works when docker isn't running.
func (c *Compose) ServiceNames(p *workspace.Project) ([]string, error) {
	envFiles := p.EnvFilePaths()

	// Relative to the project, rather than wherever orca is run from
	for i, path := range envFiles {
		envFiles[i] = inDir(p.ProjectDir, path)
	}

	spec, err := c.parser.Parse(
		[]string{inDir(p.ProjectDir, p.Config.ComposeFiles.Primary)},
		envFiles,
	)

	if err != nil {
		return nil, err
	}

	return spec.ServiceNames(), nil
}

func inDir(dir string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(dir, path)
}
