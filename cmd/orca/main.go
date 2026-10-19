package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/adamkirk/orca/internal/logging"
	"github.com/adamkirk/orca/internal/plugins"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const configFileName = "orca.yaml"
const configFileOverrideEnv = "ORCA_CONFIG_PATH"
const configPluginsPathOverrideEnv = "ORCA_PLUGINS_PATH"

// Checked in order of precedence, matching the gh cli
var githubTokenEnvs = []string{"GH_TOKEN", "GITHUB_TOKEN"}

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

// handleGroup runs a command that only groups subcommands. Cobra only rejects
// unknown subcommands for the root command, and passes them to other groups as
// arguments, so they're rejected here, with the same message.
func handleGroup(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	// Cobra prints the error, and this replaces the usage it would print after it
	cmd.SilenceUsage = true

	return unknownCommandError(cmd, args[0])
}

func unknownCommandError(cmd *cobra.Command, arg string) error {
	// Cobra's default for the root command, which is 0 (no suggestions) elsewhere
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}

	suggestions := ""
	if found := cmd.SuggestionsFor(arg); len(found) > 0 {
		suggestions = fmt.Sprintf("\n\nDid you mean this?\n\t%s\n", strings.Join(found, "\n\t"))
	}

	return fmt.Errorf(
		"%w %q for %q%s\nRun '%s --help' for usage.",
		ErrUnknownCommand,
		arg,
		cmd.CommandPath(),
		suggestions,
		cmd.CommandPath(),
	)
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

func getPluginCommandCachePath() string {
	homeDir, err := os.UserHomeDir()
	checkErr(err)

	return fmt.Sprintf("%s/.orca/cache/plugin-commands.json", homeDir)
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

func getGithubToken() string {
	for _, env := range githubTokenEnvs {
		if token := os.Getenv(env); token != "" {
			return token
		}
	}

	return ""
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

func addWorkspaceOption(cmd *cobra.Command, required bool) {
	cmd.Flags().StringP("workspace", "w", "", "The name of the workspace to run this command for.")
	checkErr(cmd.RegisterFlagCompletionFunc("workspace", completeWorkspaces))

	if required {
		cmd.MarkFlagRequired("workspace")
	}
}

func addProjectOption(cmd *cobra.Command) {
	cmd.Flags().StringP("project", "p", "", "The name of the project within the workspace to run this command for.")
	checkErr(cmd.RegisterFlagCompletionFunc("project", completeProjects))
}

// rejectDashesAfterName fails if args[0], the name of what the rest of args are
// passed to (e.g. exec's service), is followed by '--'. Everything after the
// name is passed through as it is, including a '--', which used to be needed
// before flags, and would now be passed on as an argument.
func rejectDashesAfterName(args []string, what string) error {
	if len(args) < 2 || args[1] != "--" {
		return nil
	}

	return usererr.Wrap(
		fmt.Errorf("%w after the %s", ErrUnneededDashes, what),
		fmt.Sprintf("Everything after the %s is passed on as it is, flags included, so drop the '--'!", what),
	)
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

	// Anything logged would be shown in the middle of the user's command line
	if completing {
		slog.SetDefault(slog.New(slog.DiscardHandler))
		return
	}

	h, err := logging.NewSlogHandler(cfg)

	checkErr(err)

	slog.SetDefault(slog.New(h))
}

// parsePersistentFlagsEarly parses the flags (e.g. --log-level) before cobra
// does. Plugins are loaded before cobra runs, and they need to respect these.
//
// They're parsed as cobra will parse them, for the command being run, so that
// arguments passed through to a container, as exec and ext do, aren't mistaken
// for orca's own flags. Plugin commands don't exist yet, so a line running one
// is parsed as the root command's.
func parsePersistentFlagsEarly(args []string) {
	cmd, flagArgs, err := rootCmd.Find(args)

	if err != nil {
		cmd, flagArgs = rootCmd, args
	}

	// A plugin command's flags don't exist yet, so are skipped. Restored after,
	// so cobra still rejects unknown flags.
	allowlist := cmd.FParseErrWhitelist
	cmd.FParseErrWhitelist.UnknownFlags = true
	defer func() { cmd.FParseErrWhitelist = allowlist }()

	// Any real errors will be reported when cobra parses the flags
	_ = cmd.ParseFlags(flagArgs)
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
	parsePersistentFlagsEarly(os.Args[1:])
	bootstrap()

	stopHandlingSignals := handleSignals()
	defer stopHandlingSignals()
	// Deferred after the above, so signals are still handled while plugins stop
	defer svcContainer.KillPlugins()

	// Done here rather than in init, so all builtin commands are registered first,
	// and before plugin commands are added, as their arguments may be files
	noFileCompletions(rootCmd)

	// Starting every plugin would make completion far too slow
	if !completing {
		registerPluginCommands(svcContainer.GetConfig())
		cachePluginCommands()
	}

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
