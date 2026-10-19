package main

import (
	"github.com/adamkirk/orca/internal/stack"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: `Tails logs from docker compose project.`,
	Long:  `Basically a 'docker compose logs -f', you may supply a service to tail specifically.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  handleErrors(handleLogs),
}

func init() {
	addWorkspaceOption(logsCmd, false)
	addProjectOption(logsCmd)
	rootCmd.AddCommand(logsCmd)
}

func handleLogs(cmd *cobra.Command, args []string) error {
	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")
	service := ""

	if len(args) > 0 {
		service = args[0]
	}

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.LogsQuery{
		Workspace: ws,
		Project:   project,
		Service:   service,
	})

	return err
}
