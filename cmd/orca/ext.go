package main

import (
	"github.com/adamkirk/orca/internal/controller"
	"github.com/spf13/cobra"
)

var extCmd = &cobra.Command{
	Use:   "ext",
	Short: "Execute a custom extension, defined in the project configuration",
	RunE:  handleErrors(handleExt),
}

func init() {
	addWorkspaceOption(extCmd, false)
	addProjectOption(extCmd)
	rootCmd.AddCommand(extCmd)
}

func handleExt(cmd *cobra.Command, args []string) error {
	ctrl := svcContainer.GetController()

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	// TODO: validate arguments provided
	extensionName := args[0]
	extensionArgs := args[1:]

	return ctrl.ExecuteExtension(controller.ExecuteExtensionDTO{
		Workspace: ws,
		Project:   project,
		Name:      extensionName,
		Args:      extensionArgs,
	})
}
