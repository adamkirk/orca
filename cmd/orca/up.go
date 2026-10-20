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
A disabled project given with --project is still started.

Each project's up.pre hook runs before it's started, and its up.post after. For the whole workspace, the
workspace's up.pre runs first, and its up.post last. Untrusted extensions are asked about before anything starts.`,
	RunE: handleErrors(handleUp),
}

func init() {
	addWorkspaceOption(upCmd, false)
	addServiceProjectOption(upCmd)
	addYesFlag(upCmd)

	rootCmd.AddCommand(upCmd)
}

func handleUp(cmd *cobra.Command, args []string) error {
	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.UpCommand{
		Workspace: ws,
		Project:   project,
		Yes:       mustGetBool(cmd, "yes"),
	})

	return err
}
