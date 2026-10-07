package main

import (
	"errors"

	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/tls"
	"github.com/spf13/cobra"
)

var tlsCmd = &cobra.Command{
	Use:   "tls",
	Short: "Commands to do with TLS certificates.",
	RunE:  handleGroup,
}

var tlsGenCmd = &cobra.Command{
	Use: "gen",
	Short: `Generates TLS certificates for the current workspace.
If no root certificate has been generated, one will be generated.`,
	RunE: handleErrors(handleTLSGen),
}

func init() {
	addWorkspaceOption(tlsGenCmd, false)
	tlsCmd.AddCommand(tlsGenCmd)
	rootCmd.AddCommand(tlsCmd)
}

func handleTLSGen(cmd *cobra.Command, args []string) error {
	cm := svcContainer.GetCertificateManager()
	tui := svcContainer.GetTui()

	// Resolve the workspace as other commands do, so it can come from the project
	// we're in or the current workspace, rather than requiring --workspace
	ctx, err := svcContainer.GetContextResolver().Resolve(mustGetString(cmd, "workspace"), "")

	if err != nil {
		var noWorkspaceErr common.ErrCouldNotDetermineWorkspace

		if errors.As(err, &noWorkspaceErr) {
			return tui.RecordIfError(
				"Could not determine the workspace; run this from within a project, pass --workspace, or choose one with 'orca ws switch'.",
				err,
			)
		}

		return tui.RecordIfError("Failed to determine the workspace!", err)
	}

	return cm.Generate(tls.GenerateDTO{
		WorkspaceName: ctx.Workspace.Name,
	})
}
