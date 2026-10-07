package plugins

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// BinaryPrefix is the filename prefix every plugin executable must have.
const BinaryPrefix = "orca-plugin-"

// Discover returns the paths of all plugin executables found in the given
// directories. Directories that don't exist are skipped. If the same plugin name
// appears in multiple directories, the first one wins.
func Discover(afs afero.Fs, dirs []string) ([]string, error) {
	found := []string{}
	seen := map[string]bool{}

	for _, dir := range dirs {
		entries, err := afero.ReadDir(afs, dir)

		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			return nil, err
		}

		for _, e := range entries {
			name := e.Name()

			if e.IsDir() || !strings.HasPrefix(name, BinaryPrefix) || seen[name] {
				continue
			}

			// Must be executable by someone
			if e.Mode().Perm()&0111 == 0 {
				continue
			}

			seen[name] = true
			found = append(found, filepath.Join(dir, name))
		}
	}

	return found, nil
}
