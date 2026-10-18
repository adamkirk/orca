package main

import (
	"fmt"

	"github.com/adamkirk/orca/internal/sys"
	"github.com/spf13/cobra"
)

var sysCmd = &cobra.Command{
	Use:   "sys",
	Short: "Commands for handling the installation of this tool.",
	RunE:  handleGroup,
}

var sysCheckCmd = &cobra.Command{
	Use:          "check",
	Short:        "Checks for dependencies.",
	Long:         `Checks that the required system dependencies are installed and usable.`,
	RunE:         handleErrors(handleCheck),
	SilenceUsage: true,
}

var sysInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Installs a tool system tool that is needed.",
	Long: fmt.Sprintf(`Tools are installed 'locally' rather than globally, they will be stored within %s.

The first argument must be one of: %s`, getToolsDir(), sys.AllAvailableToolsCsv()),
	RunE:         handleErrors(handleSysInstall),
	SilenceUsage: true,
}

var sysSelfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Updates this tool.",
	Long:  `By default will update to the latest available version. A specific version can be specified if a specific version is required.`,
	RunE:  handleErrors(handleSysSelfUpdate),
}

func init() {
	sysSelfUpdateCmd.Flags().String("to", "", "The version you wish to switch to. If left blank will download latest avaialable")
	sysCmd.AddCommand(sysSelfUpdateCmd)

	sysCmd.AddCommand(sysCheckCmd)
	sysCmd.AddCommand(sysInstallCmd)
	rootCmd.AddCommand(sysCmd)
}

func handleCheck(cmd *cobra.Command, _ []string) error {
	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), sys.CheckSetupQuery{})

	return err
}

func handleSysInstall(cmd *cobra.Command, args []string) error {
	tui := svcContainer.GetTui()
	if len(args) < 1 {
		return tui.RecordIfError("Must provide the name of the tool as the first argument", ErrArgumentRequired{
			Name:  "tool",
			Index: 1,
		})
	} else if len(args) > 1 {
		return tui.RecordIfError("Too many arguments provided!", ErrTooManyArguments{
			Expected: 1,
		})
	}

	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), sys.InstallCommand{
		Tool: sys.AvailableTool(args[0]),
	})

	return err
}

func handleSysSelfUpdate(cmd *cobra.Command, _ []string) error {
	to := mustGetString(cmd, "to")

	strategy := sys.VersioningStrategyLatest

	if to != "" {
		strategy = sys.VersioningStrategySpecific
	}
	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), sys.UpdateSelfCommand{
		Strategy:         strategy,
		SpecifiedVersion: to,
	})

	return err
}
