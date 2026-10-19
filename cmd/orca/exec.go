package main

import (
	"fmt"

	"github.com/adamkirk/orca/internal/stack"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <service> [command...]",
	Short: "Runs a command inside one of the containers in the environment",
	Long: `Runs a command in one of the project's services, with 'docker compose exec' if it's
running, or 'docker compose run --rm' if it isn't.

Everything after the service is passed to the container as it is, so orca's own flags
must come before it, e.g. 'orca exec -p api app ls -la'.`,
	ValidArgsFunction: firstArgOnly(completeServices),
	RunE:              handleErrors(handleExec),
}

func init() {
	addWorkspaceOption(execCmd, false)
	addProjectOption(execCmd)
	// Leave flags after the service for the command in the container
	execCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(execCmd)
}

func handleExec(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return usererr.Wrap(fmt.Errorf("%w: 'service' at position 1", ErrArgumentRequired), "Must supply a service!")
	}

	if err := rejectDashesAfterName(args, "service"); err != nil {
		return err
	}

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.ExecCommand{
		Workspace: ws,
		Project:   project,
		Service:   args[0],
		Args:      args[1:],
	})

	return err
}
