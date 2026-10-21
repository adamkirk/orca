package extension

import (
	"context"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// RunCommand runs one of the project's extensions, by name.
type RunCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
	Name      string
	Args      []string
}

func (RunCommand) MessageName() string {
	return "ext"
}

func (m *Manager) Run(ctx context.Context, cmd RunCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	run, err := m.find(execCtx, cmd.Name)

	if err != nil {
		return err
	}

	// Asking for it by name is trusting it, so it isn't asked about when a hook
	// runs it
	if err := m.trust.Trust(map[TrustKey]Trusted{run.trustKey(): trusted(run.ext)}); err != nil {
		return usererr.Wrap(err, "Failed to record the extension as trusted!")
	}

	return usererr.Wrap(m.execute(run.target, run.ext, cmd.Args, ""), fmt.Sprintf("Extension '%s' failed!", cmd.Name))
}

// find looks for the extension in the project, which extensions belong to.
func (m *Manager) find(execCtx workspace.ExecutionContext, name string) (planned, error) {
	if execCtx.Project == nil {
		return planned{}, usererr.Wrap(
			fmt.Errorf("%w: %s", ErrNoProject, name),
			"Extensions belong to projects; run it from inside a project, or pass -p.",
		)
	}

	ext, err := execCtx.Project.FindExtension(name)

	if err != nil {
		return planned{}, usererr.Wrap(err, fmt.Sprintf("There's no extension '%s' in %s!", name, execCtx.Project.Name))
	}

	return planned{target: target{ws: execCtx.Workspace, project: execCtx.Project}, ext: ext}, nil
}
