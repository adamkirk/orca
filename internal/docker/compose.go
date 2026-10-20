package docker

import (
	"fmt"
	"os"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/compose-spec/compose-go/v2/types"
)

type composeOverlayGenerator interface {
	CreateOrRetrieve(ws *workspace.Workspace, p *workspace.Project) (string, error)
}

type tui interface {
	Info(msg ...string)
	Success(msg ...string)
	NewLine()
}

type cli interface {
	Exec(cmdName string, args []string, opts ...process.ExecOpt) (err error)
}

type composeParser interface {
	Parse(paths []string, envFiles []string) (*types.Project, error)
}

type Compose struct {
	cli              cli
	tui              tui
	overlayGenerator composeOverlayGenerator
	parser           composeParser
}

func (c *Compose) getOverlay(ws *workspace.Workspace, p *workspace.Project) (string, error) {
	return c.overlayGenerator.CreateOrRetrieve(ws, p)
}

// TODO: gaurd against nil values
func (c *Compose) goToProject(p *workspace.Project) error {
	return os.Chdir(p.ProjectDir)
}

// composeProjectName is the name docker compose knows the project by.
func composeProjectName(ws *workspace.Workspace, p *workspace.Project) string {
	return fmt.Sprintf("orca-%s-%s", ws.Name, p.Name)
}

func buildBaseComposeCommand(ws *workspace.Workspace, p *workspace.Project, overlayPath string) []string {
	envArgs := []string{}

	for _, e := range p.Config.EnvFiles {
		envArgs = append(envArgs, "--env-file", e.Path)
	}

	args := []string{
		"docker",
		"compose",
		"-f",
		p.Config.ComposeFiles.Primary,
		"-f",
		overlayPath,
		"-p",
		composeProjectName(ws, p),
	}

	args = append(args, envArgs...)

	return args
}

func NewCompose(cli cli, tui tui, overlayGenerator composeOverlayGenerator, parser composeParser) *Compose {
	return &Compose{
		cli:              cli,
		tui:              tui,
		overlayGenerator: overlayGenerator,
		parser:           parser,
	}
}
