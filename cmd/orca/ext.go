package main

import (
	"fmt"

	"github.com/adamkirk/orca/internal/extension"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/spf13/cobra"
)

var extCmd = &cobra.Command{
	Use:   "ext <extension> [args...]",
	Short: "Run an extension, defined in the project or workspace config",
	Long: `Runs one of the project's extensions, or the workspace's when the project
doesn't define one with that name, or outside of any project. Any arguments are
appended to a command extension's command.

Everything after the extension's name is passed on as it is, so orca's own flags
must come before it, e.g. 'orca ext -p api artisan migrate --force'.`,
	ValidArgsFunction: firstArgOnly(completeExtensions),
	RunE:              handleErrors(handleExt),
}

func init() {
	addWorkspaceOption(extCmd, false)
	addProjectOption(extCmd)
	// Leave flags after the extension's name for the extension
	extCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(extCmd)
}

func handleExt(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return usererr.Wrap(fmt.Errorf("%w: 'extension' at position 1", ErrArgumentRequired), "Must supply an extension!")
	}

	if err := rejectDashesAfterName(args, "extension's name"); err != nil {
		return err
	}

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), extension.RunCommand{
		Workspace: ws,
		Project:   project,
		Name:      args[0],
		Args:      args[1:],
	})

	return err
}
