package main

import (
	"testing"

	"github.com/adamkirk/orca/internal/plugins"
	"github.com/adamkirk/orca/pkg/plugin"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RegisterCachedPluginCommands(t *testing.T) {
	cache := plugins.NewCommandCache(afero.NewMemMapFs(), "/home/.orca/cache/plugin-commands.json")

	require.NoError(t, cache.Write([]plugins.LoadedPlugin{{
		Name: "acme-tools",
		Commands: []plugin.CommandSpec{
			{
				Use:         "deploy",
				Short:       "Deploy to an environment",
				Flags:       []plugin.FlagSpec{{Name: "env", Type: plugin.FlagString}},
				Subcommands: []plugin.CommandSpec{{Use: "status", Short: "Show the deploy status"}},
			},
			// Taken by orca's own command, so skipped
			{Use: "up", Short: "Not orca's up"},
		},
	}}))

	svcContainer.pluginCommandCache = cache
	upShort := upCmd.Short

	t.Cleanup(func() {
		svcContainer.pluginCommandCache = nil

		if deploy, _, err := rootCmd.Find([]string{"deploy"}); err == nil && deploy != rootCmd {
			rootCmd.RemoveCommand(deploy)
		}
	})

	registerCachedPluginCommands()

	deploy, _, err := rootCmd.Find([]string{"deploy"})
	require.NoError(t, err)
	assert.Equal(t, "Deploy to an environment (plugin: acme-tools)", deploy.Short)
	assert.NotNil(t, deploy.Flags().Lookup("env"))

	status, _, err := rootCmd.Find([]string{"deploy", "status"})
	require.NoError(t, err)
	assert.Equal(t, "status", status.Name())

	up, _, err := rootCmd.Find([]string{"up"})
	require.NoError(t, err)
	assert.Equal(t, upShort, up.Short, "orca's own command wins")
}
