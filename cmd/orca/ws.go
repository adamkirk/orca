package main

import (
	"fmt"

	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/workspaces"
	"github.com/spf13/cobra"
)

var wsCmd = &cobra.Command{
	Use:   "ws",
	Short: "Commands related to managing workspaces.",
	RunE:  handleGroup,
}

var wsSwitchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch to another workspace",
	RunE:  handleErrors(handleWsSwitch),
}

var wsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialises a workspace.",
	Long: `Initialises a new workspace with orca. Either a local directory or a git repository can be supplied.
When using a local directory, will locate an orca workspace config and clone the relevant projects.
When using a git url, will clone the given repository, and then clone any other required repositories based on an orca workspace file within the repo.`,
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
The project option allows you to clone only a specific project.`,
	RunE: handleErrors(handleWsClone),
}

func init() {
	wsCmd.AddCommand(wsSwitchCmd)

	wsInitCmd.Flags().StringP("source", "s", getWorkingDir(), "The directory of the workspace configuration.")
	wsInitCmd.Flags().StringP("target", "t", getWorkingDirParent(), "The directory to store the workspace projects.")
	wsInitCmd.Flags().StringP("config", "c", workspaces.DefaultWorkspaceFileName, "The name of the workspace config file within the source.")
	wsCmd.AddCommand(wsInitCmd)

	wsCmd.AddCommand(wsLsCmd)
	wsCmd.AddCommand(wsCurrentCmd)
	wsCmd.AddCommand(wsClearCurrentCmd)

	wsCloneCmd.Flags().StringP("target", "t", "", `The directory in which to clone the project(s). 
If multiple projects are being cloned, then it will place them in {target}/{repo name}.
If a single project is being clone then it will be cloned into {target}.`)
	addWorkspaceOption(wsCloneCmd, false)
	addProjectOption(wsCloneCmd)
	wsCmd.AddCommand(wsCloneCmd)

	rootCmd.AddCommand(wsCmd)
}

func handleWsSwitch(cmd *cobra.Command, args []string) error {
	tui := svcContainer.GetTui()

	if len(args) < 1 {
		tui.Error("Must supply a workspace!")

		return common.ErrUnknownWorkspace{
			Name: "",
		}
	}

	manager := svcContainer.GetConfig()

	if err := manager.SwitchWorkspace(args[0]); err != nil {
		return tui.RecordIfError("Failed to switch workspace!", err)
	}

	tui.Success(fmt.Sprintf("Switched to %s", args[0]))

	return nil
}

func handleWsInit(cmd *cobra.Command, args []string) error {
	manager := svcContainer.GetWorkspaceManager()

	source := mustGetString(cmd, "source")
	target := mustGetString(cmd, "target")
	configFile := mustGetString(cmd, "config")

	return manager.Initialise(workspaces.InitialiseDTO{
		SourceDirectory:   source,
		WorkspaceFileName: configFile,
		Into:              target,
	})
}

func handleWsLs(cmd *cobra.Command, args []string) error {
	manager := svcContainer.GetWorkspaceManager()

	return manager.Ls(workspaces.LsDTO{})
}

func handleWsCurrent(cmd *cobra.Command, args []string) error {
	manager := svcContainer.GetWorkspaceManager()

	return manager.ShowCurrent(workspaces.ShowCurrentDTO{})
}

func handleWsClearCurrent(cmd *cobra.Command, args []string) error {
	manager := svcContainer.GetWorkspaceManager()

	return manager.ClearCurrent(workspaces.ClearCurrentDTO{})
}

func handleWsClone(cmd *cobra.Command, args []string) error {
	manager := svcContainer.GetWorkspaceManager()

	name := mustGetString(cmd, "workspace")
	projectName := mustGetString(cmd, "project")
	to := mustGetString(cmd, "target")

	return manager.Clone(workspaces.CloneDTO{
		WorkspaceName: name,
		Project:       projectName,
		To:            to,
	})
}
