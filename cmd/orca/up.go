package main

import (
	"github.com/adamkirk/orca/internal/stack"
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
	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.UpCommand{
		Workspace: ws,
		Project:   project,
	})

	return err
}
