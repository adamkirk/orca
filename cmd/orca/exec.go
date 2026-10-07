package main

import (
	"github.com/adamkirk/orca/internal/controller"
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
	ctrl := svcContainer.GetController()

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")
	service := mustGetString(cmd, "service")

	return ctrl.ExecOrRun(controller.ExecDTO{
		Workspace: ws,
		Project:   project,
		Service:   service,
		Args:      args,
	})
}
