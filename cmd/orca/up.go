package main

import (
	"github.com/adamkirk/orca/internal/controller"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Starts the workspace or project.",
	Long:  `...TBD...`,
	RunE:  handleErrors(handleUp),
}

func init() {
	addWorkspaceOption(upCmd, false)
	addProjectOption(upCmd)

	rootCmd.AddCommand(upCmd)
}

func handleUp(cmd *cobra.Command, args []string) error {
	ctrl := svcContainer.GetController()

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	return ctrl.Up(controller.UpDTO{
		Workspace: ws,
		Project:   project,
	})
}
