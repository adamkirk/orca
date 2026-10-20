package main

import (
	"errors"
	"testing"

	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func Test_IsCompletionRequest(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "completion", args: []string{"orca", "__complete", "up", ""}, want: true},
		{name: "completion without descriptions", args: []string{"orca", "__completeNoDesc", "up", ""}, want: true},
		{name: "a command", args: []string{"orca", "up"}},
		{name: "generating the script", args: []string{"orca", "completion", "zsh"}},
		{name: "no arguments", args: []string{"orca"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isCompletionRequest(tt.args))
		})
	}
}

func Test_ToCompletions(t *testing.T) {
	choices := []workspace.Choice{
		{Value: "api"},
		{Value: "admin", Description: "The admin panel"},
		{Value: "web"},
	}

	tests := []struct {
		name       string
		err        error
		toComplete string
		want       []cobra.Completion
	}{
		{
			name: "everything, when nothing is typed",
			want: []cobra.Completion{"api", "admin\tThe admin panel", "web"},
		},
		{
			name:       "only those starting with what's typed",
			toComplete: "a",
			want:       []cobra.Completion{"api", "admin\tThe admin panel"},
		},
		{
			name:       "nothing matches",
			toComplete: "x",
		},
		{
			name: "nothing on an error",
			err:  errors.New("invalid config"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, directive := toCompletions(choices, tt.err, tt.toComplete)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func Test_FirstArgOnly(t *testing.T) {
	f := firstArgOnly(func(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return []cobra.Completion{"acme"}, cobra.ShellCompDirectiveNoFileComp
	})

	got, _ := f(&cobra.Command{}, nil, "")
	assert.Equal(t, []cobra.Completion{"acme"}, got)

	got, directive := f(&cobra.Command{}, []string{"acme"}, "")
	assert.Empty(t, got)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func Test_NoFileCompletions(t *testing.T) {
	custom := func(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	}

	root := &cobra.Command{Use: "orca"}
	grp := &cobra.Command{Use: "grp"}
	leaf := &cobra.Command{Use: "leaf"}
	completes := &cobra.Command{Use: "completes", ValidArgsFunction: custom}
	static := &cobra.Command{Use: "static", ValidArgs: []string{"a"}}

	grp.AddCommand(leaf)
	root.AddCommand(grp, completes, static)

	noFileCompletions(root)

	for _, cmd := range []*cobra.Command{root, grp, leaf} {
		_, directive := cmd.ValidArgsFunction(cmd, nil, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive, cmd.Name())
	}

	_, directive := completes.ValidArgsFunction(completes, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveDefault, directive, "keeps its own")
	assert.Nil(t, static.ValidArgsFunction, "static args are left to cobra")
}

func Test_WithoutGiven(t *testing.T) {
	choices := []workspace.Choice{{Value: "api"}, {Value: "web"}, {Value: "admin"}}

	assert.Equal(t, []workspace.Choice{{Value: "api"}, {Value: "admin"}}, withoutGiven(choices, []string{"web"}))
	assert.Equal(t, []workspace.Choice{{Value: "api"}}, withoutGiven([]workspace.Choice{{Value: "api"}}, nil))
}
