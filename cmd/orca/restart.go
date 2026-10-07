package main

import (
	"github.com/adamkirk/orca/internal/controller"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Alias for running down && up.",
	Long:  `...TBD...`,
	RunE:  handleErrors(handleRestart),
}

func init() {
	addWorkspaceOption(restartCmd, false)
	addProjectOption(restartCmd)

	rootCmd.AddCommand(restartCmd)
}

func handleRestart(cmd *cobra.Command, args []string) error {
	ctrl := svcContainer.GetController()

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	err := ctrl.Down(controller.DownDTO{
		Workspace: ws,
		Project:   project,
	})

	if err != nil {
		return err
	}

	return ctrl.Up(controller.UpDTO{
		Workspace: ws,
		Project:   project,
	})
}
