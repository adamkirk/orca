package plugins

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Discover(t *testing.T) {
	fs := afero.NewMemMapFs()

	require.NoError(t, afero.WriteFile(fs, "/a/orca-plugin-hello", []byte{}, 0755))
	require.NoError(t, afero.WriteFile(fs, "/a/orca-plugin-not-executable", []byte{}, 0644))
	require.NoError(t, afero.WriteFile(fs, "/a/something-else", []byte{}, 0755))
	require.NoError(t, fs.MkdirAll("/a/orca-plugin-dir", 0755))
	require.NoError(t, afero.WriteFile(fs, "/b/orca-plugin-hello", []byte{}, 0755))
	require.NoError(t, afero.WriteFile(fs, "/b/orca-plugin-other", []byte{}, 0755))

	found, err := Discover(fs, []string{"/missing", "/a", "/b"})

	require.NoError(t, err)
	assert.Equal(t, []string{"/a/orca-plugin-hello", "/b/orca-plugin-other"}, found)
}

func Test_DiscoverNoDirs(t *testing.T) {
	found, err := Discover(afero.NewMemMapFs(), []string{})

	require.NoError(t, err)
	assert.Empty(t, found)
}
