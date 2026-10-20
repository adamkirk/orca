package main

import (
	"github.com/adamkirk/orca/internal/stack"
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
	addServiceProjectOption(restartCmd)

	rootCmd.AddCommand(restartCmd)
}

func handleRestart(cmd *cobra.Command, args []string) error {
	b := svcContainer.GetBus()

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	_, err := b.Dispatch(cmd.Context(), stack.DownCommand{
		Workspace: ws,
		Project:   project,
	})

	if err != nil {
		return err
	}

	_, err = b.Dispatch(cmd.Context(), stack.UpCommand{
		Workspace: ws,
		Project:   project,
	})

	return err
}
