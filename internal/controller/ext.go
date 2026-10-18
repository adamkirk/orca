package controller

import (
	"context"
	"errors"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
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
func (c *Controller) executeExtensionInService(cmd ExecuteExtensionCommand, execCtx common.ExecutionContext, ext common.Extension) error {
	cmdArgs := strings.Split(ext.Command, " ")

	if len(cmd.Args) > 0 {
		cmdArgs = append(cmdArgs, cmd.Args...)
	} else {
		cmdArgs = append(cmdArgs, ext.DefaultArgs...)
	}

	isRunning, err := c.compose.IsSvcRunning(execCtx.Workspace, execCtx.Project, ext.Service)

	if err != nil {
		return err
	}

	if !isRunning {
		return c.compose.Run(execCtx.Workspace, execCtx.Project, ext.Service, cmdArgs)
	}

	return c.compose.Exec(execCtx.Workspace, execCtx.Project, ext.Service, cmdArgs)
}

func (c *Controller) ExecuteExtension(ctx context.Context, cmd ExecuteExtensionCommand) error {
	execCtx, err := common.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	if execCtx.Project == nil {
		return common.ErrInvalidExecutionContext{
			Msg: "Extensions must be executed within a project context!",
		}
	}

	ext, err := execCtx.Project.FindExtension(cmd.Name)

	if err != nil {
		return c.tui.RecordIfError("Extension does not exist in project context.", err)
	}

	if ext.Service != "" {
		return c.executeExtensionInService(cmd, execCtx, ext)
	}

	return c.tui.RecordIfError("Extension must have a service defined", errors.New("alternative is not yet implemented"))
}
