package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/adamkirk/orca/pkg/plugin"
	"github.com/spf13/afero"
)

// CachedPlugin is a plugin's commands, as stored in the CommandCache.
type CachedPlugin struct {
	Name     string
	Commands []plugin.CommandSpec
}

// CommandCache stores the commands each plugin provides, so that shell
// completion can offer them without starting any plugins, which would be far
// too slow. It's only a cache: orca rewrites it whenever it runs a command, so
// it's always safe to delete.
type CommandCache struct {
	fs   afero.Fs
	path string
}

func NewCommandCache(fs afero.Fs, path string) *CommandCache {
	return &CommandCache{
		fs:   fs,
		path: path,
	}
}

// Write replaces the cache with the commands of the given plugins, so a plugin
// that's no longer loaded is removed from it.
func (c *CommandCache) Write(loaded []LoadedPlugin) error {
	cached := make([]CachedPlugin, len(loaded))

	for i, p := range loaded {
		cached[i] = CachedPlugin{Name: p.Name, Commands: p.Commands}
	}

	data, err := json.Marshal(cached)

	if err != nil {
		return fmt.Errorf("encoding the plugin command cache: %w", err)
	}

	if err := c.fs.MkdirAll(filepath.Dir(c.path), 0700); err != nil {
		return fmt.Errorf("creating the cache directory: %w", err)
	}

	// Written in full, then moved into place, so a completion running at the
	// same time never reads half of it
	tmp := fmt.Sprintf("%s.%d.tmp", c.path, os.Getpid())

	if err := afero.WriteFile(c.fs, tmp, data, 0600); err != nil {
		return fmt.Errorf("writing the plugin command cache: %w", err)
	}

	if err := c.fs.Rename(tmp, c.path); err != nil {
		_ = c.fs.Remove(tmp)

		return fmt.Errorf("writing the plugin command cache: %w", err)
	}

	return nil
}

// Read returns the cached plugins. There's no cache until orca has run a
// command, in which case there are none.
func (c *CommandCache) Read() ([]CachedPlugin, error) {
	data, err := afero.ReadFile(c.fs, c.path)

	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading the plugin command cache: %w", err)
	}

	var cached []CachedPlugin

	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("decoding the plugin command cache: %w", err)
	}

	return cached, nil
}
