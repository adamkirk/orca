package plugins

import (
	"fmt"
	"strconv"

	"github.com/adamkirk/orca/pkg/plugin"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// BuildCommands converts a plugin's command specs into cobra commands that call
// back into the plugin when run.
func BuildCommands(provider plugin.CommandProvider, specs []plugin.CommandSpec) []*cobra.Command {
	return buildCommands(provider, specs, nil)
}

func buildCommands(provider plugin.CommandProvider, specs []plugin.CommandSpec, parentPath []string) []*cobra.Command {
	cmds := make([]*cobra.Command, len(specs))

	for i, spec := range specs {
		cmd := &cobra.Command{
			Use:   spec.Use,
			Short: spec.Short,
			Long:  spec.Long,
		}

		// Copy so sibling commands don't share a backing array
		path := append(append([]string{}, parentPath...), cmd.Name())

		for _, f := range spec.Flags {
			addFlag(cmd, f)
		}

		if len(spec.Subcommands) > 0 {
			cmd.RunE = func(cmd *cobra.Command, _ []string) error {
				return cmd.Help()
			}
			cmd.AddCommand(buildCommands(provider, spec.Subcommands, path)...)
		} else {
			cmd.RunE = executeFunc(provider, path)
		}

		cmds[i] = cmd
	}

	return cmds
}

func addFlag(cmd *cobra.Command, f plugin.FlagSpec) {
	switch f.Type {
	case plugin.FlagBool:
		def, _ := strconv.ParseBool(f.Default)
		cmd.Flags().BoolP(f.Name, f.Shorthand, def, f.Usage)
	default:
		cmd.Flags().StringP(f.Name, f.Shorthand, f.Default, f.Usage)
	}
}

func executeFunc(provider plugin.CommandProvider, path []string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		flags := map[string]string{}

		// Only the flags the plugin declared, not orca's persistent flags
		cmd.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
			if f.Name == "help" {
				return
			}

			flags[f.Name] = f.Value.String()
		})

		code, err := provider.Execute(cmd.Context(), plugin.ExecuteRequest{
			CommandPath: path,
			Args:        args,
			Flags:       flags,
		})

		if err != nil || code != 0 {
			// Orca prints plugin errors itself, without cobra's prefix and usage
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			return ExitError{Code: code, Err: err}
		}

		return nil
	}
}

// ExitError is returned when a plugin command fails, so orca can exit with the
// plugin's exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	return fmt.Sprintf("plugin command exited with code %d", e.Code)
}

func (e ExitError) Unwrap() error {
	return e.Err
}
