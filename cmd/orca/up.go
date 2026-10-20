package main

import (
	"github.com/adamkirk/orca/internal/stack"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Starts the workspace or project.",
	Long: `Starts the project, or with no project, every enabled project in the workspace, after the projects they require.
A project that an enabled project requires is started even if it's disabled, with a warning.
A disabled project given with --project is still started.`,
	RunE: handleErrors(handleUp),
}

func init() {
	addWorkspaceOption(upCmd, false)
	addServiceProjectOption(upCmd)

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
