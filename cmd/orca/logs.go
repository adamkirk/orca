package main

import (
	"github.com/adamkirk/orca/internal/controller"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: `Tails logs from docker compose project.`,
	Long:  `Basically a 'docker compose logs -f', you may supply a service to tail specifically.`,
	RunE:  handleErrors(handleLogs),
}

func init() {
	addWorkspaceOption(logsCmd, false)
	addProjectOption(logsCmd)
	addServiceOption(logsCmd, false)
	rootCmd.AddCommand(logsCmd)
}

func handleLogs(cmd *cobra.Command, args []string) error {
	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")
	service := mustGetString(cmd, "service")

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), controller.LogsQuery{
		Workspace: ws,
		Project:   project,
		Service:   service,
	})

	return err
}
