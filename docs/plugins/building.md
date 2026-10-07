# Building a plugin

This guide walks through writing a plugin in Go, using orca's plugin SDK. The SDK handles the gRPC protocol, so a plugin only has to describe its commands and run them.

The finished plugin adds an `orca greet` command, with subcommands, flags and arguments.

## Prerequisites

- Go 1.27 or later.
- A version of orca with plugin support.

The SDK lives in orca's own module, at [`github.com/adamkirk/orca/pkg/plugin`](https://github.com/adamkirk/orca/tree/main/pkg/plugin). A complete, minimal plugin is in [`examples/plugins/hello`](https://github.com/adamkirk/orca/tree/main/examples/plugins/hello).

## 1. Create the module

```sh
$ mkdir orca-plugin-greet && cd orca-plugin-greet
$ go mod init github.com/acme/orca-plugin-greet
$ go get github.com/adamkirk/orca@latest
```

## 2. Write the plugin

A plugin implements the `plugin.CommandProvider` interface, which has two methods:

- `Commands` describes the commands the plugin adds.
- `Execute` runs one of them.

`main` hands your implementation to `plugin.Serve`, and that's all it does.

```go title="main.go"
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/adamkirk/orca/pkg/plugin"
)

func main() {
	plugin.Serve(&greeter{})
}

type greeter struct{}

func (g *greeter) Commands() ([]plugin.CommandSpec, error) {
	return []plugin.CommandSpec{
		{
			Use:   "greet",
			Short: "Greets people.",
			// A command with subcommands is a group: running it on its own
			// shows its help.
			Subcommands: []plugin.CommandSpec{
				{
					Use:   "hello [name...]",
					Short: "Says hello.",
					Flags: []plugin.FlagSpec{
						{Name: "shout", Shorthand: "s", Usage: "Say it loudly.", Type: plugin.FlagBool},
					},
				},
				{
					Use:   "goodbye",
					Short: "Says goodbye.",
					Flags: []plugin.FlagSpec{
						{Name: "to", Usage: "Who to say goodbye to.", Type: plugin.FlagString, Default: "world"},
					},
				},
			},
		},
	}, nil
}

func (g *greeter) Execute(ctx context.Context, req plugin.ExecuteRequest) (int, error) {
	// CommandPath identifies the command being run, from the top level down
	switch strings.Join(req.CommandPath, " ") {
	case "greet hello":
		if len(req.Args) == 0 {
			return 2, errors.New("give me at least one name")
		}

		msg := "hello " + strings.Join(req.Args, " and ")

		// Flag values are always strings
		if req.Flags["shout"] == "true" {
			msg = strings.ToUpper(msg) + "!"
		}

		fmt.Println(msg)
	case "greet goodbye":
		fmt.Printf("goodbye %s\n", req.Flags["to"])
	default:
		return 1, fmt.Errorf("unknown command: %v", req.CommandPath)
	}

	return 0, nil
}
```

## 3. Build and install it

The binary's name must start with `orca-plugin-`, and it must be in one of orca's [plugin directories](./index.md#plugin-directories):

```sh
$ go build -o ~/.orca/plugins/orca-plugin-greet .
```

## 4. Try it

```sh
$ orca greet --help
Greets people.

Usage:
  orca greet [flags]
  orca greet [command]

Available Commands:
  goodbye     Says goodbye.
  hello       Says hello.
...

$ orca greet hello alice bob
hello alice and bob

$ orca greet hello -s alice
HELLO ALICE!

$ orca greet goodbye --to everyone
goodbye everyone

$ orca greet hello
give me at least one name
$ echo $?
2
```

Rebuild the binary after each change; orca picks up the new version the next time it runs.

## Reference

### `CommandSpec`

Describes one command. Orca turns these into real commands, so they get help output, flag parsing and validation for free.

| Field | Description |
| ----- | ----------- |
| `Use` | One-line usage. The first word is the command's name, the rest is shown in help, e.g. `"hello [name...]"`. |
| `Short` | A short description, shown in command lists. |
| `Long` | A longer description, shown in the command's own help. |
| `Flags` | The command's flags, see [`FlagSpec`](#flagspec). |
| `Subcommands` | Child commands. A command with subcommands is a group: it just shows help, and only its children are executed. |

A top-level command's name must not clash with one of orca's own commands, or another plugin's; if it does, it's skipped. `help` and `completion` are also reserved.

### `FlagSpec`

| Field | Description |
| ----- | ----------- |
| `Name` | The long name, e.g. `to` for `--to`. |
| `Shorthand` | Optional one-letter name, e.g. `s` for `-s`. |
| `Usage` | Help text. |
| `Type` | `plugin.FlagString` (the default) or `plugin.FlagBool`. |
| `Default` | The default value, as a string, e.g. `"world"` or `"true"`. |

### `ExecuteRequest`

What `Execute` receives:

| Field | Description |
| ----- | ----------- |
| `CommandPath` | The names of the commands from the plugin's top-level command down to the one being run, e.g. `["greet", "hello"]`. |
| `Args` | Positional arguments. |
| `Flags` | Every flag the command declares, keyed by `Name`, including ones the user didn't set (with their defaults). Values are strings; bools are `"true"` or `"false"`. Orca's own flags, like `--log-level`, aren't included. |

### Return values

`Execute` returns an exit code and an error:

- `0, nil`: success.
- A non-zero code: orca exits with that code.
- An error: orca prints it, and exits with the code you returned, or `1` if you returned `0`.

## Output and input

Anything the plugin writes to stdout and stderr is passed through to the user's terminal, so print as you normally would. Plugins can't read from stdin yet, so they can't prompt the user.

## Tips

### Keep `Commands` fast

Orca starts every plugin and calls `Commands` on every run, even for unrelated commands like `orca up`. Return a fixed list, and do expensive setup in `Execute`.

### Debugging

- Run orca with `--log-level debug` to see each plugin being started, and why one failed to load.
- [`orca plugins ls`](../CLI/orca_plugins_ls.md) shows whether your plugin loaded, and which commands it provided.
- Running the plugin binary directly just prints *"This binary is a plugin. These are not meant to be executed directly."* That's expected: only orca can start it.
- Logging to stderr from `Execute` is the simplest way to trace what's happening.

### Testing

`Commands` and `Execute` are ordinary methods, so test them directly, without orca. Keep the logic for each command in its own function and test that.

### Distributing

Plugins are plain binaries, so cross-compile them for your team's platforms (`GOOS=linux GOARCH=amd64 go build ...`) and share them however suits you. For example, commit them to a shared repository that everyone adds to [`plugins.dirs`](./index.md#plugin-directories).

## Lifecycle

You don't need to manage the plugin's process yourself:

- Orca starts the plugin when it runs, and stops it before exiting, including when the user presses Ctrl-C.
- If orca is killed without warning, the SDK notices that its parent process has gone and exits the plugin within a fraction of a second, so it's never left running.

`Execute` is given a `context.Context`. Pass it to anything long-running, so work stops if the connection to orca is lost.
