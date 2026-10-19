package main

import (
	"github.com/adamkirk/orca/internal/stack"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Runs a command inside one of the containers in the environment",
	Long:  `...TBD...`,
	RunE:  handleErrors(handleExec),
}

func init() {
	addWorkspaceOption(execCmd, false)
	addProjectOption(execCmd)
	addServiceOption(execCmd, true)
	rootCmd.AddCommand(execCmd)
}

func handleExec(cmd *cobra.Command, args []string) error {
	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")
	service := mustGetString(cmd, "service")

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.ExecCommand{
		Workspace: ws,
		Project:   project,
		Service:   service,
		Args:      args,
	})

	return err
}
