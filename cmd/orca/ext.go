package main

import (
	"github.com/adamkirk/orca/internal/stack"
	"github.com/spf13/cobra"
)

var extCmd = &cobra.Command{
	Use:               "ext <extension> [args...]",
	Short:             "Execute a custom extension, defined in the project configuration",
	ValidArgsFunction: firstArgOnly(completeExtensions),
	RunE:              handleErrors(handleExt),
}

func init() {
	addWorkspaceOption(extCmd, false)
	addProjectOption(extCmd)
	rootCmd.AddCommand(extCmd)
}

func handleExt(cmd *cobra.Command, args []string) error {
	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	// TODO: validate arguments provided
	extensionName := args[0]
	extensionArgs := args[1:]

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.ExecuteExtensionCommand{
		Workspace: ws,
		Project:   project,
		Name:      extensionName,
		Args:      extensionArgs,
	})

	return err
}
