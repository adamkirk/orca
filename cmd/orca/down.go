package main

import (
	"github.com/adamkirk/orca/internal/stack"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stops the workspace or project.",
	Long: `Stops the project, or with no project, every project in the workspace that's enabled, required by an enabled project, or still running, before the projects they require.

Each project's down.pre hook runs before it's stopped, and its down.post after. For the whole workspace, the
workspace's down.pre runs first, and its down.post last. Untrusted extensions are asked about before anything stops.`,
	RunE: handleErrors(handleDown),
}

func init() {
	addWorkspaceOption(downCmd, false)
	addServiceProjectOption(downCmd)
	addYesFlag(downCmd)

	rootCmd.AddCommand(downCmd)
}

func handleDown(cmd *cobra.Command, args []string) error {
	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.DownCommand{
		Workspace: ws,
		Project:   project,
		Yes:       mustGetBool(cmd, "yes"),
	})

	return err
}
