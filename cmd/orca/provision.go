package main

import (
	"github.com/adamkirk/orca/internal/controller"
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
	c := svcContainer.GetController()

	return c.Provision(controller.ProvisionDTO{})
}
