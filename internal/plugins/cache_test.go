package plugins

import (
	"testing"

	"github.com/adamkirk/orca/pkg/plugin"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cachePath = "/home/.orca/cache/plugin-commands.json"

func Test_CommandCache_Write(t *testing.T) {
	deploy := plugin.CommandSpec{
		Use:   "deploy",
		Short: "Deploy to an environment",
		Flags: []plugin.FlagSpec{{Name: "env", Shorthand: "e", Type: plugin.FlagString, Default: "staging"}},
		Subcommands: []plugin.CommandSpec{
			{Use: "status", Short: "Show the deploy status"},
		},
	}

	t.Run("stores each plugin's commands", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		cache := NewCommandCache(fs, cachePath)

		require.NoError(t, cache.Write([]LoadedPlugin{
			{Name: "acme-tools", Path: "/plugins/orca-plugin-acme-tools", Commands: []plugin.CommandSpec{deploy}},
		}))

		got, err := cache.Read()

		require.NoError(t, err)
		assert.Equal(t, []CachedPlugin{{Name: "acme-tools", Commands: []plugin.CommandSpec{deploy}}}, got)

		files, err := afero.ReadDir(fs, "/home/.orca/cache")
		require.NoError(t, err)
		assert.Len(t, files, 1, "no temporary file is left behind")
	})

	t.Run("replaces what was there, so removed plugins disappear", func(t *testing.T) {
		cache := NewCommandCache(afero.NewMemMapFs(), cachePath)

		require.NoError(t, cache.Write([]LoadedPlugin{{Name: "acme-tools", Commands: []plugin.CommandSpec{deploy}}}))
		require.NoError(t, cache.Write(nil))

		got, err := cache.Read()

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("fails if it can't be written", func(t *testing.T) {
		cache := NewCommandCache(afero.NewReadOnlyFs(afero.NewMemMapFs()), cachePath)

		assert.Error(t, cache.Write(nil))
	})
}

func Test_CommandCache_Read(t *testing.T) {
	t.Run("no cache yet", func(t *testing.T) {
		got, err := NewCommandCache(afero.NewMemMapFs(), cachePath).Read()

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("an unreadable cache", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, cachePath, []byte("{not json"), 0600))

		_, err := NewCommandCache(fs, cachePath).Read()

		assert.ErrorContains(t, err, "decoding the plugin command cache")
	})
}
