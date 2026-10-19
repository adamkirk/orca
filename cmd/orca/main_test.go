package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func Test_HandleGroup(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr string
	}{
		{
			name: "without arguments, shows the help",
			args: []string{"grp"},
		},
		{
			name:      "an unknown subcommand is an error",
			args:      []string{"grp", "nosuch"},
			expectErr: "unknown command \"nosuch\" for \"test grp\"\nRun 'test grp --help' for usage.",
		},
		{
			name:      "suggests similar subcommands",
			args:      []string{"grp", "inti"},
			expectErr: "unknown command \"inti\" for \"test grp\"\n\nDid you mean this?\n\tinit\n\nRun 'test grp --help' for usage.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &cobra.Command{Use: "test"}
			grp := &cobra.Command{Use: "grp", RunE: handleGroup}
			grp.AddCommand(&cobra.Command{Use: "init", Run: func(*cobra.Command, []string) {}})
			root.AddCommand(grp)

			out := &bytes.Buffer{}
			root.SetOut(out)
			root.SetErr(out)
			root.SetArgs(tt.args)

			err := root.Execute()

			if tt.expectErr == "" {
				assert.NoError(t, err)
				assert.Contains(t, out.String(), "Available Commands:")
				return
			}

			assert.ErrorIs(t, err, ErrUnknownCommand)
			assert.EqualError(t, err, tt.expectErr)
			assert.NotContains(t, out.String(), "Available Commands:")
		})
	}
}
