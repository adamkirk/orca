package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var utilCmd = &cobra.Command{
	Use:   "util",
	Short: "Utility commands, mainly helper type commands for internal use.",
	RunE:  handleGroup,
}

var utilGenDocsCmd = &cobra.Command{
	Use:   "gen-docs",
	Short: "Generates markdown documentation for the CLI.",
	RunE:  handleErrors(handleUtilGenDocs),
}

func init() {
	utilCmd.AddCommand(utilGenDocsCmd)

	rootCmd.AddCommand(utilCmd)
}

func handleUtilGenDocs(cmd *cobra.Command, _ []string) error {
	err := doc.GenMarkdownTree(rootCmd, "./docs/CLI")

	return err
}
