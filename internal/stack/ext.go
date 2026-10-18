package stack

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

type ExecuteExtensionCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
	Name      string
	Args      []string
}

func (ExecuteExtensionCommand) MessageName() string {
	return "ext"
}

// TODO: de-dupe this by reusing the ExecOrRun method
func (m *Manager) executeExtensionInService(cmd ExecuteExtensionCommand, execCtx workspace.ExecutionContext, ext workspace.Extension) error {
	cmdArgs := strings.Split(ext.Command, " ")

	if len(cmd.Args) > 0 {
		cmdArgs = append(cmdArgs, cmd.Args...)
	} else {
		cmdArgs = append(cmdArgs, ext.DefaultArgs...)
	}

	isRunning, err := m.compose.IsSvcRunning(execCtx.Workspace, execCtx.Project, ext.Service)

	if err != nil {
		return err
	}

	if !isRunning {
		return m.compose.Run(execCtx.Workspace, execCtx.Project, ext.Service, cmdArgs)
	}

	return m.compose.Exec(execCtx.Workspace, execCtx.Project, ext.Service, cmdArgs)
}

func (m *Manager) ExecuteExtension(ctx context.Context, cmd ExecuteExtensionCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	if execCtx.Project == nil {
		return fmt.Errorf("%w: %s", workspace.ErrInvalidExecutionContext, "Extensions must be executed within a project context!")
	}

	ext, err := execCtx.Project.FindExtension(cmd.Name)

	if err != nil {
		return usererr.Wrap(err, "Extension does not exist in project context.")
	}

	if ext.Service != "" {
		return m.executeExtensionInService(cmd, execCtx, ext)
	}

	return usererr.Wrap(errors.New("alternative is not yet implemented"), "Extension must have a service defined")
}
