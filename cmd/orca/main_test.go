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

func Test_ParsePersistentFlagsEarly(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "before the command",
			args: []string{"--log-level", "debug", "up"},
			want: "debug",
		},
		{
			name: "after the command",
			args: []string{"up", "--log-level", "debug"},
			want: "debug",
		},
		{
			name: "before exec's service",
			args: []string{"exec", "--log-level", "debug", "app", "sh"},
			want: "debug",
		},
		{
			name: "after exec's service, it's the container's",
			args: []string{"exec", "app", "sh", "--log-level", "debug"},
		},
		{
			name: "a plugin's command, which doesn't exist yet",
			args: []string{"some-plugin", "--plugin-flag", "--log-level", "debug"},
			want: "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := rootCmd.PersistentFlags().Lookup("log-level")
			before := flag.Value.String()

			t.Cleanup(func() {
				flag.Value.Set(before)
				flag.Changed = false
			})

			parsePersistentFlagsEarly(tt.args)

			if tt.want == "" {
				assert.False(t, flag.Changed)
				return
			}

			assert.Equal(t, tt.want, flag.Value.String())
		})
	}
}

func Test_RejectDashesAfterName(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "just the name", args: []string{"app"}},
		{name: "a command", args: []string{"app", "ls", "-la"}},
		{name: "dashes later on are the command's", args: []string{"app", "ls", "--", "-la"}},
		{name: "dashes straight after the name", args: []string{"app", "--", "ls"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rejectDashesAfterName(tt.args, "service")

			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}

			assert.ErrorIs(t, err, ErrUnneededDashes)
			assert.EqualError(t, err, "Everything after the service is passed on as it is, flags included, so drop the '--'!: unneeded '--' after the service")
		})
	}
}
