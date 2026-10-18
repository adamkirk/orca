package main

import (
	"github.com/adamkirk/orca/internal/provision"
	"github.com/spf13/cobra"
)

var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Run any provisioning scripts for the given project",
	RunE:  handleErrors(handleProvision),
}

func init() {
	rootCmd.AddCommand(provisionCmd)
}

func handleProvision(cmd *cobra.Command, args []string) error {
	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), provision.RunCommand{})

	return err
}
