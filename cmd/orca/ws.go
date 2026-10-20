package main

import (
	"fmt"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/cobra"
)

var wsCmd = &cobra.Command{
	Use:   "ws",
	Short: "Commands related to managing workspaces.",
	RunE:  handleGroup,
}

var wsSwitchCmd = &cobra.Command{
	Use:               "switch <workspace>",
	Short:             "Switch to another workspace",
	ValidArgsFunction: firstArgOnly(completeWorkspaces),
	RunE:              handleErrors(handleWsSwitch),
}

var wsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialises a workspace.",
	Long: `Initialises a new workspace with orca. Either a local directory or a GitHub repository can be supplied.
When using a local directory, will locate an orca workspace config and clone the relevant projects.
When using a GitHub repository (--repo owner/repo), will check the workspace config on its default branch (or --branch) is valid, clone the repository over SSH into the target, and then clone any other required repositories based on that workspace config.
Projects with a 'repository.branch' in the workspace config are cloned on that branch. Every branch is checked before anything is cloned.
Set GH_TOKEN or GITHUB_TOKEN to use a private repository.

Once everything is cloned, each project's clone.post hook runs, then the workspace's init.post. Any extension
they run that you haven't trusted yet is listed first, and you're asked to trust it.

It's safe to run again, in the same place: it clones whatever's missing, and runs every hook again. That's
how to finish a setup that failed part-way, once the problem is fixed.`,
	RunE: handleErrors(handleWsInit),
}

var wsLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "Lists all available workspaces.",
	RunE:  handleErrors(handleWsLs),
}

var wsCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Shows the current workspace",
	RunE:  handleErrors(handleWsCurrent),
}

var wsClearCurrentCmd = &cobra.Command{
	Use:   "clear-current",
	Short: "Deselects the current workspace at a global level",
	RunE:  handleErrors(handleWsClearCurrent),
}

var wsCloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clones all the projects required for this workspace.",
	Long: `Based on the workspace config will clone each repository required by the workspace.
This can be run at any time to clone any projects that have not already been cloned.
The project option allows you to clone only a specific project.

Then the clone.post hook of every project it was asked for runs, whether it was cloned now or before.`,
	RunE: handleErrors(handleWsClone),
}

var wsProjectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Lists the workspace's projects, and whether each is enabled, cloned and running.",
	Long: `Lists the projects in the workspace, with whether each is enabled in your preferences file, cloned, and running.
Running is '?' when docker can't be asked.`,
	RunE: handleErrors(handleWsProjects),
}

var wsEnableCmd = &cobra.Command{
	Use:   "enable <project>...",
	Short: "Enables projects, so workspace-wide commands run them.",
	Long: `Enables the projects in your preferences file, so 'up', 'down' and 'g --all' include them.
It doesn't start them.`,
	ValidArgsFunction: completeDisabledProjects,
	RunE:              handleErrors(handleWsEnable),
}

var wsDisableCmd = &cobra.Command{
	Use:   "disable <project>...",
	Short: "Disables projects, so workspace-wide commands skip them.",
	Long: `Disables the projects in your preferences file, so 'up', 'down' and 'g --all' skip them, unless an enabled project requires them.
Disabled projects are still cloned, and can still be run with --project.
Any of them that are running are stopped, unless a running project requires them, or --orphan is given.`,
	ValidArgsFunction: completeEnabledProjects,
	RunE:              handleErrors(handleWsDisable),
}

func init() {
	wsCmd.AddCommand(wsSwitchCmd)

	addWorkspaceOption(wsProjectsCmd, false)
	wsCmd.AddCommand(wsProjectsCmd)

	addWorkspaceOption(wsEnableCmd, false)
	wsCmd.AddCommand(wsEnableCmd)

	wsDisableCmd.Flags().Bool("orphan", false, "Leaves the projects running, rather than stopping them.")
	addWorkspaceOption(wsDisableCmd, false)
	wsCmd.AddCommand(wsDisableCmd)

	wsInitCmd.Flags().StringP("source", "s", getWorkingDir(), "The directory of the workspace configuration.")
	wsInitCmd.Flags().StringP("target", "t", getWorkingDirParent(), "The directory to store the workspace projects. Defaults to the current directory when --repo is used.")
	wsInitCmd.Flags().StringP("repo", "r", "", "The github repository to clone the workspace from, in the format 'owner/repo'.")
	wsInitCmd.Flags().StringP("config", "c", workspace.DefaultWorkspaceFileName, "The name of the workspace config file within the source.")
	wsInitCmd.Flags().StringP("branch", "b", "", "The branch of the --repo repository to read the workspace config from, and to clone. Defaults to the repository's default branch. Each project's branch comes from its 'repository.branch'.")
	addHookFlags(wsInitCmd)
	wsInitCmd.MarkFlagsMutuallyExclusive("repo", "source")
	wsInitCmd.MarkFlagsMutuallyExclusive("branch", "source")
	wsCmd.AddCommand(wsInitCmd)

	wsCmd.AddCommand(wsLsCmd)
	wsCmd.AddCommand(wsCurrentCmd)
	wsCmd.AddCommand(wsClearCurrentCmd)

	wsCloneCmd.Flags().StringP("target", "t", "", `The directory in which to clone the project(s). 
If multiple projects are being cloned, then it will place them in {target}/{repo name}.
If a single project is being clone then it will be cloned into {target}.`)
	addHookFlags(wsCloneCmd)
	addWorkspaceOption(wsCloneCmd, false)
	addProjectOption(wsCloneCmd)
	wsCmd.AddCommand(wsCloneCmd)

	rootCmd.AddCommand(wsCmd)
}

func handleWsSwitch(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return usererr.Wrap(fmt.Errorf("%w: 'workspace' at position 1", ErrArgumentRequired), "Must supply a workspace!")
	}

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.SwitchCommand{
		Workspace: args[0],
	})

	return err
}

func handleWsInit(cmd *cobra.Command, args []string) error {
	source := mustGetString(cmd, "source")
	target := mustGetString(cmd, "target")
	repo := mustGetString(cmd, "repo")
	configFile := mustGetString(cmd, "config")
	branch := mustGetString(cmd, "branch")

	// When cloning the workspace repo, it lands inside the target alongside the
	// other projects, so default to the current directory rather than its parent
	if repo != "" && !cmd.Flags().Changed("target") {
		target = getWorkingDir()
	}

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.InitialiseCommand{
		SourceRepo:        repo,
		Branch:            branch,
		SourceDirectory:   source,
		WorkspaceFileName: configFile,
		Into:              target,
		SkipHooks:         mustGetBool(cmd, "skip-hooks"),
		Yes:               mustGetBool(cmd, "yes"),
	})

	return err
}

func handleWsLs(cmd *cobra.Command, args []string) error {
	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.LsQuery{})

	return err
}

func handleWsCurrent(cmd *cobra.Command, args []string) error {
	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.ShowCurrentQuery{})

	return err
}

func handleWsClearCurrent(cmd *cobra.Command, args []string) error {
	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.ClearCurrentCommand{})

	return err
}

func handleWsClone(cmd *cobra.Command, args []string) error {
	name := mustGetString(cmd, "workspace")
	projectName := mustGetString(cmd, "project")
	to := mustGetString(cmd, "target")

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.CloneCommand{
		WorkspaceName: name,
		Project:       projectName,
		To:            to,
		SkipHooks:     mustGetBool(cmd, "skip-hooks"),
		Yes:           mustGetBool(cmd, "yes"),
	})

	return err
}

func handleWsProjects(cmd *cobra.Command, args []string) error {
	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.ProjectsQuery{
		Workspace: mustGetString(cmd, "workspace"),
	})

	return err
}

func handleWsEnable(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return usererr.Wrap(fmt.Errorf("%w: 'project' at position 1", ErrArgumentRequired), "Must supply at least one project!")
	}

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.EnableCommand{
		Workspace: mustGetString(cmd, "workspace"),
		Projects:  args,
	})

	return err
}

func handleWsDisable(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return usererr.Wrap(fmt.Errorf("%w: 'project' at position 1", ErrArgumentRequired), "Must supply at least one project!")
	}

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), workspace.DisableCommand{
		Workspace: mustGetString(cmd, "workspace"),
		Projects:  args,
		Orphan:    mustGetBool(cmd, "orphan"),
	})

	return err
}

// addHookFlags adds the flags for commands that run hooks after cloning.
func addHookFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("skip-hooks", false, "Clone without running any hooks.")
	cmd.Flags().BoolP("yes", "y", false, "Run every extension the hooks run without asking, for this run only: it doesn't record them as trusted.")
}
