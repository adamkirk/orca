package main

import (
	"github.com/adamkirk/orca/internal/controller"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Commands to aid in debugging or understanding whats happening.",
	RunE:  handleGroup,
}

var debugShowComposeConfigCmd = &cobra.Command{
	Use:   "show-compose-config",
	Short: `Shows the full generated compose config that will be used for this project.`,
	Long: `Similar to the 'docker compose config' command (it uses this under the 
hood), it will show you the resulting config after all files are merged together, 
including any relevant values from environment variables, or profile changes etc.`,
	RunE: handleErrors(handleDebugShowComposeConfig),
}

var debugShowComposeCommandCmd = &cobra.Command{
	Use:   "show-compose-command",
	Short: `Shows the compose command being used for this project.`,
	Long:  `Can be used to run generic commands via: ` + "`$(orca debug show-compose-command) ps`",
	RunE:  handleErrors(handleDebugShowComposeCommand),
}

func init() {
	addWorkspaceOption(debugShowComposeConfigCmd, false)
	addProjectOption(debugShowComposeConfigCmd)
	debugCmd.AddCommand(debugShowComposeConfigCmd)

	addWorkspaceOption(debugShowComposeCommandCmd, false)
	addProjectOption(debugShowComposeCommandCmd)
	debugCmd.AddCommand(debugShowComposeCommandCmd)
	rootCmd.AddCommand(debugCmd)
}

func handleDebugShowComposeConfig(cmd *cobra.Command, args []string) error {
	ctrl := svcContainer.GetController()

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	return ctrl.ShowComposeConfig(controller.ShowComposeConfigDTO{
		Workspace: ws,
		Project:   project,
	})
}

func handleDebugShowComposeCommand(cmd *cobra.Command, args []string) error {
	ctrl := svcContainer.GetController()

	ws := mustGetString(cmd, "workspace")
	project := mustGetString(cmd, "project")

	return ctrl.ShowComposeCommand(controller.ShowComposeCommandDTO{
		Workspace: ws,
		Project:   project,
	})
}
