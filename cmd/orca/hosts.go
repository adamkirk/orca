package main

import (
	"github.com/adamkirk/orca/internal/hosts"
	"github.com/spf13/cobra"
)

var hostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: `Shows all the required hosts entries for the workspace.`,
	RunE:  handleErrors(handleHosts),
}

func init() {
	addWorkspaceOption(hostsCmd, false)
	rootCmd.AddCommand(hostsCmd)
}

func handleHosts(cmd *cobra.Command, args []string) error {
	ws := mustGetString(cmd, "workspace")

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), hosts.SyncCommand{
		Workspace: ws,
	})

	return err
}
