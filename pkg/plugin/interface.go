// Package plugin is the SDK for writing orca plugins in Go.
//
// A plugin is a standalone binary named orca-plugin-<name>, placed in one of
// orca's plugin directories. Its main function only needs to call Serve with an
// implementation of CommandProvider:
//
//	func main() {
//		plugin.Serve(&myProvider{})
//	}
package plugin

import "context"

type FlagType int

const (
	FlagString FlagType = iota
	FlagBool
)

type FlagSpec struct {
	Name      string
	Shorthand string
	Usage     string
	Type      FlagType
	// Default value, encoded as a string regardless of type, e.g. "true" for a
	// bool flag.
	Default string
}

type CommandSpec struct {
	// Use is the one-line usage message, the first word is the command name.
	Use         string
	Short       string
	Long        string
	Flags       []FlagSpec
	Subcommands []CommandSpec
}

type ExecuteRequest struct {
	// CommandPath is the names of the commands from the plugin's top-level
	// command down to the one being executed, e.g. ["k8s", "ctx"].
	CommandPath []string
	Args        []string
	// Flags holds every flag value for the command keyed by name, encoded as
	// strings.
	Flags map[string]string
}

// CommandProvider is implemented by plugins that contribute CLI commands to orca.
type CommandProvider interface {
	// Commands returns the command tree to register with orca.
	Commands() ([]CommandSpec, error)
	// Execute runs one of the commands returned by Commands. Anything written to
	// stdout/stderr is forwarded to the user's terminal.
	Execute(ctx context.Context, req ExecuteRequest) (exitCode int, err error)
}
