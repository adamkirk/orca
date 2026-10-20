package main

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/adamkirk/orca/internal/config"
	"github.com/adamkirk/orca/internal/plugins"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Commands for managing plugins.",
	RunE:  handleGroup,
}

var pluginsLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "Lists the loaded plugins and the commands they provide.",
	RunE:  handleErrors(handlePluginsLs),
}

func init() {
	pluginsCmd.AddCommand(pluginsLsCmd)
	rootCmd.AddCommand(pluginsCmd)
}

// Commands cobra adds itself during Execute, so they aren't found by Find yet.
var reservedCommandNames = []string{"help", "completion"}

// registerPluginCommands starts every discovered plugin and adds its commands to
// the root command. This must run after all init functions, so that every
// builtin command is registered before we check for name collisions.
func registerPluginCommands(cfg *config.Config) {
	mgr := svcContainer.GetPluginManager()
	logger := mgr.Logger()

	dirs := append([]string{getPluginsDir()}, cfg.GetPluginDirs()...)

	paths, err := plugins.Discover(svcContainer.GetFs(), dirs)

	if err != nil {
		logger.Warn("failed to discover plugins", "err", err)
		return
	}

	for _, p := range mgr.Load(paths) {
		addPluginCommands(p.Name, plugins.BuildCommands(p.Provider, p.Commands), logger)
	}
}

// registerCachedPluginCommands adds the commands from the plugin command cache
// to the root command, for completion, which can't start the plugins. They're
// only completed, never run, so they have no plugin to call.
func registerCachedPluginCommands() {
	cached, err := svcContainer.GetPluginCommandCache().Read()

	if err != nil {
		slog.Debug("failed to read the plugin command cache", "err", err)
		return
	}

	for _, p := range cached {
		cmds := plugins.BuildCommands(nil, p.Commands)

		// Shown next to the command, so it's clear where it comes from
		for _, cmd := range cmds {
			cmd.Short = strings.TrimSpace(fmt.Sprintf("%s (plugin: %s)", cmd.Short, p.Name))
		}

		addPluginCommands(p.Name, cmds, slog.Default())
	}
}

// addPluginCommands adds a plugin's commands to the root command, skipping any
// whose name is already taken.
func addPluginCommands(plugin string, cmds []*cobra.Command, logger *slog.Logger) {
	for _, cmd := range cmds {
		if cmd.Name() == "" || commandNameTaken(cmd.Name()) {
			logger.Warn("plugin command conflicts with an existing command, skipping", "plugin", plugin, "command", cmd.Name())
			continue
		}

		rootCmd.AddCommand(cmd)
	}
}

// cachePluginCommands stores the commands of the plugins that loaded, so that
// completion can offer them without starting the plugins. It's only a cache,
// so failing to write it never fails the command.
func cachePluginCommands() {
	err := svcContainer.GetPluginCommandCache().Write(svcContainer.GetPluginManager().Loaded())

	if err != nil {
		slog.Debug("failed to write the plugin command cache", "err", err)
	}
}

func commandNameTaken(name string) bool {
	for _, reserved := range reservedCommandNames {
		if name == reserved {
			return true
		}
	}

	for _, c := range rootCmd.Commands() {
		if c.Name() == name || c.HasAlias(name) {
			return true
		}
	}

	return false
}

func handlePluginsLs(cmd *cobra.Command, _ []string) error {
	loaded := svcContainer.GetPluginManager().Loaded()

	if len(loaded) == 0 {
		fmt.Println("No plugins loaded.")
		return nil
	}

	for _, p := range loaded {
		names := make([]string, len(p.Commands))

		for i, c := range p.Commands {
			names[i] = (&cobra.Command{Use: c.Use}).Name()
		}

		fmt.Printf("%s (%s)\n  commands: %s\n", p.Name, p.Path, strings.Join(names, ", "))
	}

	return nil
}
