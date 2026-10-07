package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the current version you're using.",
	RunE:  handleErrors(handleVersion),
}

func init() {
	versionCmd.Flags().Bool("short", false, "Show only the version, excluding commit and date information.")
	rootCmd.AddCommand(versionCmd)
}

func handleVersion(cmd *cobra.Command, _ []string) error {
	short, err := cmd.Flags().GetBool("short")

	if err != nil {
		return err
	}

	if short {
		fmt.Println(version)
		return nil
	}

	fmt.Printf("%s (%s@%s)\n", version, commit, date)

	return nil
}
