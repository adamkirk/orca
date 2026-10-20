package main

import (
	"fmt"

	"github.com/adamkirk/orca/internal/extension"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Commands to do with the hooks that run extensions around operations.",
	RunE:  handleGroup,
}

var hooksRunCmd = &cobra.Command{
	Use:   "run <operation>.<pre|post>",
	Short: "Run a hook's extensions by hand, e.g. to set a project up again.",
	Long: `Runs every extension in one of the project's hooks, in order, e.g. 'orca hooks run clone.post -p api'.

Outside of a project, or for a hook only the workspace has (init.post), it runs the workspace's hook.
It stops at the first extension that fails. Extensions that are new, or have changed since you trusted
them, are listed first, and you're asked to trust them.`,
	ValidArgsFunction: firstArgOnly(completeHooks),
	RunE:              handleErrors(handleHooksRun),
}

func init() {
	addWorkspaceOption(hooksRunCmd, false)
	addProjectOption(hooksRunCmd)
	hooksRunCmd.Flags().BoolP("yes", "y", false, "Run every extension without asking, for this run only: it doesn't record them as trusted.")
	hooksCmd.AddCommand(hooksRunCmd)
	rootCmd.AddCommand(hooksCmd)
}

func handleHooksRun(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return usererr.Wrap(fmt.Errorf("%w: 'hook' at position 1", ErrArgumentRequired), "Must supply one hook, e.g. 'clone.post'!")
	}

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), extension.RunHookCommand{
		Workspace: mustGetString(cmd, "workspace"),
		Project:   mustGetString(cmd, "project"),
		Hook:      args[0],
		Yes:       mustGetBool(cmd, "yes"),
	})

	return err
}
