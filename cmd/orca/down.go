package main

import (
	"github.com/adamkirk/orca/internal/controller"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stops the workspace or project.",
	Long:  `...TBD...`,
	RunE:  handleErrors(handleDown),
}

func init() {
	addWorkspaceOption(downCmd, false)
	addProjectOption(downCmd)

	rootCmd.AddCommand(downCmd)
}

func handleDown(cmd *cobra.Command, args []string) error {
	ctrl := svcContainer.GetController()

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	return ctrl.Down(controller.DownDTO{
		Workspace: ws,
		Project:   project,
	})
}
