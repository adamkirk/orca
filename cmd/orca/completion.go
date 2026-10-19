package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/adamkirk/orca/internal/stack"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/cobra"
)

// completing is true when orca was run by a shell's completion script, to list
// what can be typed next, rather than by the user. Completion must be quick and
// silent: it never starts plugins, logs, or prints errors.
var completing = isCompletionRequest(os.Args)

func isCompletionRequest(args []string) bool {
	if len(args) < 2 {
		return false
	}

	return args[1] == cobra.ShellCompRequestCmd || args[1] == cobra.ShellCompNoDescRequestCmd
}

// noFileCompletions stops cmd, and every command below it, from offering file
// names for arguments, unless they complete their arguments themselves. Orca's
// arguments are names, never files, so a typo shouldn't fall back to them.
func noFileCompletions(cmd *cobra.Command) {
	if cmd.ValidArgsFunction == nil && len(cmd.ValidArgs) == 0 {
		cmd.ValidArgsFunction = cobra.NoFileCompletions
	}

	for _, c := range cmd.Commands() {
		noFileCompletions(c)
	}
}

// toCompletions converts choices into cobra's completions, keeping those that
// start with what's been typed. A failure offers nothing, rather than files.
func toCompletions(choices []workspace.Choice, err error, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if err != nil {
		slog.Debug("completion failed", "err", err)

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []cobra.Completion

	for _, c := range choices {
		if !strings.HasPrefix(c.Value, toComplete) {
			continue
		}

		if c.Description == "" {
			completions = append(completions, c.Value)
			continue
		}

		completions = append(completions, cobra.CompletionWithDesc(c.Value, c.Description))
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// firstArgOnly wraps a completion function so that it only completes the first
// argument, and offers nothing after it.
func firstArgOnly(f cobra.CompletionFunc) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return f(cmd, args, toComplete)
	}
}

func completeWorkspaces(cmd *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	choices, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.WorkspaceChoicesQuery{})

	return toCompletions(choices, err, toComplete)
}

func completeProjects(cmd *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	// Not every command with --project has --workspace
	ws, _ := cmd.Flags().GetString("workspace")

	choices, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.ProjectChoicesQuery{
		Workspace: ws,
	})

	return toCompletions(choices, err, toComplete)
}

func completeServices(cmd *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	choices, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.ServiceChoicesQuery{
		Workspace: mustGetString(cmd, "workspace"),
		Project:   mustGetString(cmd, "project"),
	})

	return toCompletions(choices, err, toComplete)
}

func completeExtensions(cmd *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	choices, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.ExtensionChoicesQuery{
		Workspace: mustGetString(cmd, "workspace"),
		Project:   mustGetString(cmd, "project"),
	})

	return toCompletions(choices, err, toComplete)
}
