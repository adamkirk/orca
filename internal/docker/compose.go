package docker

import (
	"fmt"
	"os"

	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/process"
)

type composeOverlayGenerator interface {
	CreateOrRetrieve(ws *common.Workspace, p *common.Project) (string, error)
}

type tui interface {
	Info(msg ...string)
	Error(msg ...string)
	Success(msg ...string)
	NewLine()
	RecordIfError(msg string, err error) error
}

type cli interface {
	Exec(cmdName string, args []string, opts ...process.ExecOpt) (err error)
}

type Compose struct {
	cli              cli
	tui              tui
	overlayGenerator composeOverlayGenerator
}

func (c *Compose) getOverlay(ws *common.Workspace, p *common.Project) (string, error) {
	return c.overlayGenerator.CreateOrRetrieve(ws, p)
}

// TODO: gaurd against nil values
func (c *Compose) goToProject(p *common.Project) error {
	return os.Chdir(p.ProjectDir)
}

func buildBaseComposeCommand(ws *common.Workspace, p *common.Project, overlayPath string) []string {
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
		fmt.Sprintf("orca-%s-%s", ws.Name, p.Name),
	}

	args = append(args, envArgs...)

	return args
}

func NewCompose(cli cli, tui tui, overlayGenerator composeOverlayGenerator) *Compose {
	return &Compose{
		cli:              cli,
		tui:              tui,
		overlayGenerator: overlayGenerator,
	}
}
