package main

import (
	"fmt"

	"github.com/adamkirk/orca/internal/git"
	"github.com/spf13/cobra"
)

var gCmd = &cobra.Command{
	Use:   "g",
	Short: "Commands related to git.",
	RunE:  handleGroup,
}

var gCoCmd = &cobra.Command{
	Use:   "co",
	Short: "Checkout a branch for a git repository.",
	Long: `Searches for a branch with the name provided as an argument. If a single branch is
found, it will be checked out. If multiple are found will provide a list to select from.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		rebase, _ := cmd.Flags().GetBool("rebase")
		pull, _ := cmd.Flags().GetBool("pull")

		if rebase && !pull {
			return fmt.Errorf("--rebase can only be used with --pull")
		}

		return nil
	},
	RunE: handleErrors(handleGCo),
}

var gBranchesCmd = &cobra.Command{
	Use:   "branches",
	Short: "Searches branches for the git repository.",
	Long: `Will search for any branches containing the given search term (case-insensitive).
If no search term is given, will list all branches.`,
	RunE: handleErrors(handleGBranches),
}

var gPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pulls a branch from origin.",
	Long:  `Will pull the currently checked out branch. Use --all to pull the current branch in every project in the workspace.`,
	RunE:  handleErrors(handleGPull),
}

var gRbiCmd = &cobra.Command{
	Use:   "rbi",
	Short: "Run an interactive rebase.",
	Long:  `Starts an interactive rebase, for the number of commits required.`,
	RunE:  handleErrors(handleGRebaseInteractively),
}

var gPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Pushes the branch to origin.",
	Long:  `Pushes the current branch to origin, using the current branches name as the target on the origin. Use --all to push the current branch in every project in the workspace.`,
	RunE:  handleErrors(handleGPush),
}

var gUndoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Removes commits from the branch.",
	Long:  `This will (destructively) remove commits from the current branch. The number of commits to remove is defined by the 'number' option.`,
	RunE:  handleErrors(handleGUndo),
}

var gStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Shows the checked out branch for projects.",
	Long:  `Shows a table of the currently checked out branch. Use --all to show every project in the workspace.`,
	RunE:  handleErrors(handleGStatus),
}

var gLoglCmd = &cobra.Command{
	Use:   "logl",
	Short: "Shows the last X commits.",
	Long:  `Only shows the short commit sha and subject of each commit for the last X commits on the current branch.`,
	RunE:  handleErrors(handleGLogl),
}

func init() {
	gCoCmd.Flags().Bool("pull", false, "Pulls the branch from origin after checking it out.")
	gCoCmd.Flags().BoolP("all", "a", false, "Checks out the chosen branch in each project in the workspace.")
	gCoCmd.Flags().BoolP("create", "b", false, "Creates the branch if it doesn't exist.")
	gCoCmd.Flags().BoolP("rebase", "r", false, "Passes --rebase to git pull, only valid with --pull.")
	addWorkspaceOption(gCoCmd, false)
	addProjectOption(gCoCmd)
	gCoCmd.MarkFlagsMutuallyExclusive("all", "project")
	gCoCmd.MarkFlagsMutuallyExclusive("create", "pull")
	gCmd.AddCommand(gCoCmd)

	gCmd.AddCommand(gBranchesCmd)

	gRbiCmd.Flags().IntP("number", "n", 2, "The number of commits to include in the interactive rebase.")
	gCmd.AddCommand(gRbiCmd)

	gPushCmd.Flags().BoolP("force", "f", false, "Whether force push the branch.")
	gPushCmd.Flags().BoolP("all", "a", false, "Pushes the current branch in each project in the workspace.")
	addWorkspaceOption(gPushCmd, false)
	addProjectOption(gPushCmd)
	gPushCmd.MarkFlagsMutuallyExclusive("all", "project")
	gCmd.AddCommand(gPushCmd)

	gUndoCmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt, and just delete them. #YOLO")
	gUndoCmd.Flags().IntP("number", "n", 1, "The number of commits to remove from the branch.")
	gCmd.AddCommand(gUndoCmd)

	gLoglCmd.Flags().IntP("number", "n", 10, "The number of commits to remove from the branch.")
	gCmd.AddCommand(gLoglCmd)

	gStatusCmd.Flags().BoolP("all", "a", false, "Shows the current branch for each project in the workspace.")
	addWorkspaceOption(gStatusCmd, false)
	addProjectOption(gStatusCmd)
	gStatusCmd.MarkFlagsMutuallyExclusive("all", "project")
	gCmd.AddCommand(gStatusCmd)

	gPullCmd.Flags().BoolP("all", "a", false, "Pulls the current branch in each project in the workspace.")
	gPullCmd.Flags().BoolP("rebase", "r", false, "Passes --rebase to git pull.")
	addWorkspaceOption(gPullCmd, false)
	addProjectOption(gPullCmd)
	gPullCmd.MarkFlagsMutuallyExclusive("all", "project")
	gCmd.AddCommand(gPullCmd)

	rootCmd.AddCommand(gCmd)
}

func handleGCo(cmd *cobra.Command, args []string) error {
	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	g := svcContainer.GetGit()

	searchTerm := ""

	if len(args) > 0 {
		searchTerm = args[0]
	}

	allProjects := mustGetBool(cmd, "all")
	shouldCreate := mustGetBool(cmd, "create")
	shouldPull := mustGetBool(cmd, "pull")
	shouldRebase := mustGetBool(cmd, "rebase")

	err := g.Checkout(git.CheckoutDTO{
		Name:        searchTerm,
		AllProjects: allProjects,
		Workspace:   ws,
		Project:     project,
		Pull:        shouldPull,
		Rebase:      shouldRebase,
		Create:      shouldCreate,
	})

	if err != nil {
		return err
	}

	return nil
}

func handleGBranches(cmd *cobra.Command, args []string) error {
	g := svcContainer.GetGit()

	searchTerm := ""

	if len(args) > 0 {
		searchTerm = args[0]
	}

	return g.ShowBranches(git.SearchBranchesDTO{
		Search: searchTerm,
	})
}

func handleGRebaseInteractively(cmd *cobra.Command, args []string) error {
	g := svcContainer.GetGit()

	amount := mustGetInt(cmd, "number")

	return g.RebaseInteractively(git.RebaseInteractivelyDTO{
		Amount: amount,
	})
}

func handleGPush(cmd *cobra.Command, args []string) error {
	g := svcContainer.GetGit()

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")
	allProjects := mustGetBool(cmd, "all")
	force := mustGetBool(cmd, "force")

	return g.Push(git.PushDTO{
		AllProjects: allProjects,
		Workspace:   ws,
		Project:     project,
		Force:       force,
	})
}

func handleGUndo(cmd *cobra.Command, args []string) error {
	g := svcContainer.GetGit()

	autoConfirm := mustGetBool(cmd, "yes")
	amount := mustGetInt(cmd, "number")

	return g.UndoLastXCommits(git.UndoLastXCommitsDTO{
		Amount:           amount,
		SkipConfirmation: autoConfirm,
	})
}

func handleGLogl(cmd *cobra.Command, args []string) error {
	g := svcContainer.GetGit()

	amount := mustGetInt(cmd, "number")

	return g.Logl(git.LoglDTO{
		Amount: amount,
	})
}

func handleGStatus(cmd *cobra.Command, args []string) error {
	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")
	allProjects := mustGetBool(cmd, "all")

	g := svcContainer.GetGit()

	return g.Status(git.StatusDTO{
		AllProjects: allProjects,
		Workspace:   ws,
		Project:     project,
	})
}

func handleGPull(cmd *cobra.Command, args []string) error {
	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")
	allProjects := mustGetBool(cmd, "all")
	rebase := mustGetBool(cmd, "rebase")

	g := svcContainer.GetGit()

	return g.Pull(git.PullDTO{
		AllProjects: allProjects,
		Workspace:   ws,
		Project:     project,
		Rebase:      rebase,
	})
}
