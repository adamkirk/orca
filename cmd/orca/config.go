package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Commands for managing configuration",
	RunE:  handleGroup,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the config",
	RunE:  handleErrors(handleConfigShow),
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the path to config being used.",
	RunE:  handleErrors(handleConfigPath),
}

func init() {
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}

func handleConfigShow(cmd *cobra.Command, _ []string) error {
	contents, err := yaml.Marshal(svcContainer.GetConfig().GetRuntimeConfig())

	checkErr(err)

	fmt.Print(string(contents))

	return nil
}

func handleConfigPath(cmd *cobra.Command, _ []string) error {
	fmt.Println(getConfigFilePath())

	return nil
}
