package main

import (
	"fmt"
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
		for _, cmd := range plugins.BuildCommands(p.Provider, p.Commands) {
			if cmd.Name() == "" || commandNameTaken(cmd.Name()) {
				logger.Warn("plugin command conflicts with an existing command, skipping", "plugin", p.Name, "command", cmd.Name())
				continue
			}

			rootCmd.AddCommand(cmd)
		}
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
