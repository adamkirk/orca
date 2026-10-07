package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/adamkirk/orca/internal/logging"
	"github.com/adamkirk/orca/internal/plugins"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const configFileName = "orca.yaml"
const configFileOverrideEnv = "ORCA_CONFIG_PATH"
const configToolsPathOverrideEnv = "ORCA_TOOLS_PATH"
const configPluginsPathOverrideEnv = "ORCA_PLUGINS_PATH"

type runEHandlerFunc func(cmd *cobra.Command, args []string) error

var (
	version string = "dev"
	commit  string = "none"
	date    string = "unknown"
)

var svcContainer *services = &services{}

var rootCmd = &cobra.Command{
	Use:   "orca",
	Short: "Orca is used to orchestrate complex development environments with docker compose.",
	Long: `This can be used to manage multiple projects in different repositories
so that they can be started/stopped as a singular unit, and provides common
utilities to interact with services form anywhere on the host.`,
	RunE: handleGroup,
}

func handleGroup(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}

func getToolsDir() string {
	homeDir, err := os.UserHomeDir()
	checkErr(err)

	configFile := fmt.Sprintf("%s/.orca/bin", homeDir)

	if override, found := os.LookupEnv(configToolsPathOverrideEnv); found {
		configFile = override
	}

	return configFile
}

func getPluginsDir() string {
	homeDir, err := os.UserHomeDir()
	checkErr(err)

	dir := fmt.Sprintf("%s/.orca/plugins", homeDir)

	if override, found := os.LookupEnv(configPluginsPathOverrideEnv); found {
		dir = override
	}

	return dir
}

func getOverlayDir() string {
	homeDir, err := os.UserHomeDir()
	checkErr(err)

	return fmt.Sprintf("%s/.orca/overlays", homeDir)
}

func getTLSDir() string {
	homeDir, err := os.UserHomeDir()
	checkErr(err)

	dir := fmt.Sprintf("%s/.orca/tls", homeDir)

	return dir
}

func getConfigFilePath() string {
	homeDir, err := os.UserHomeDir()
	checkErr(err)

	configFile := fmt.Sprintf("%s/.orca/%s", homeDir, configFileName)

	if override, found := os.LookupEnv(configFileOverrideEnv); found {
		configFile = override
	}

	return configFile
}

func getWorkingDir() string {
	wd, err := os.Getwd()

	checkErr(err)

	return wd
}

func getWorkingDirParent() string {
	wd := getWorkingDir()

	dir := path.Dir(fmt.Sprintf("%s/../", wd))

	_, err := os.Stat(dir)

	checkErr(err)

	return dir
}

func init() {
	cfg := svcContainer.GetConfig()

	err := cfg.LoadOrCreate()
	checkErr(err)

	cobra.OnInitialize(bootstrap)

	// Persistent flags
	rootCmd.PersistentFlags().String("log-level", cfg.GetLoggingLevel(), "Log level to use, one of: debug, info, warn, error, none. Defaults to none, as most errors are already surfaced anyway.")
	rootCmd.PersistentFlags().String("log-format", cfg.GetLoggingFormat(), "log format to use")

	checkErr(viper.BindPFlag("logging.level", rootCmd.PersistentFlags().Lookup("log-level")))
	checkErr(viper.BindPFlag("logging.format", rootCmd.PersistentFlags().Lookup("log-format")))
}

func addServiceOption(cmd *cobra.Command, required bool) {
	cmd.Flags().StringP("service", "s", "", "The service within the project to exec into")

	if required {
		cmd.MarkFlagRequired("service")
	}
}

func addWorkspaceOption(cmd *cobra.Command, required bool) {
	cmd.Flags().StringP("workspace", "w", "", "The name of the workspace to run this command for.")

	if required {
		cmd.MarkFlagRequired("workspace")
	}
}

func addProjectOption(cmd *cobra.Command) {
	cmd.Flags().StringP("project", "p", "", "The name of the project within the workspace to run this command for.")
}

func bootstrap() {
	cfg := svcContainer.GetConfig()

	// Tell viper to replace . in nested path with underscores
	// e.g. logging.level becomes LOGGING_LEVEL
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetEnvPrefix("orca")
	viper.AutomaticEnv()

	err := viper.Unmarshal(cfg.GetRuntimeConfig())
	checkErr(err)

	h, err := logging.NewSlogHandler(cfg)

	checkErr(err)

	slog.SetDefault(slog.New(h))
}

// parsePersistentFlagsEarly parses the root command's persistent flags (e.g.
// --log-level) before cobra does, ignoring any other flags. Plugins are loaded
// before cobra runs, and they need to respect these.
func parsePersistentFlagsEarly() {
	fs := pflag.NewFlagSet("early", pflag.ContinueOnError)
	fs.ParseErrorsAllowlist.UnknownFlags = true
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	// Shares the underlying flags, so parsed values land on the root command
	fs.AddFlagSet(rootCmd.PersistentFlags())

	// Any real errors will be reported when cobra parses the flags
	_ = fs.Parse(os.Args[1:])
}

func main() {
	os.Exit(run())
}

// run is the entrypoint, it returns the exit code rather than exiting so that
// deferred cleanup always runs. See exit.go for how plugin processes are kept
// from outliving orca.
func run() int {
	// Configure logging from flags/env before plugins are started, cobra will run
	// bootstrap again once it has parsed the flags itself.
	parsePersistentFlagsEarly()
	bootstrap()

	stopHandlingSignals := handleSignals()
	defer stopHandlingSignals()
	// Deferred after the above, so signals are still handled while plugins stop
	defer svcContainer.KillPlugins()

	// Done here rather than in init, so all builtin commands are registered first
	registerPluginCommands(svcContainer.GetConfig())

	err := rootCmd.Execute()

	// Any error is likely from plugins being stopped by the signal handler, which
	// is about to exit with this code anyway.
	if code := signalExitCode.Load(); code != 0 {
		return int(code)
	}

	if err == nil {
		return 0
	}

	var exitErr plugins.ExitError
	if errors.As(err, &exitErr) {
		fmt.Fprintln(os.Stderr, err)

		if exitErr.Code != 0 {
			return exitErr.Code
		}
	}

	return 1
}
