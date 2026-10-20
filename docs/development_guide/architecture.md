# Architecture

This page covers how the code is organised, and the conventions to follow when changing it. The [command bus](./command_bus.md) and [errors](./errors.md) pages go into those two parts in more detail.

## How a command runs

Every command that does something goes through the same path:

```
cobra handler (cmd/orca)          reads flags, builds a message, dispatches it
  └─ bus.Dispatch                 internal/bus
       ├─ LoggingMiddleware       logs the dispatch at debug level
       ├─ ExecutionContextMiddleware   resolves the workspace/project, if the message has a target
       ├─ Before hooks
       ├─ handler                 a method on a feature package's Manager
       └─ After hooks
  └─ reportError                  shows any error to the user, once (cmd/orca/exit.go)
```

`cmd/orca` is the only package that knows about cobra, flags or how services are wired together. Everything under `internal/` works with messages, `context.Context` and its own small interfaces.

## Packages

Each package is one of three kinds, and named after what it provides, not its layer. There's no `common`, `util`, `controller` or `repository` package.

### Domain

| Package | Contents |
| ------- | -------- |
| `internal/workspace` | Everything about workspaces: the model (`Workspace`, `Project`, `Extension`...), loading their config (`Repository`), working out which workspace and project a command runs against (`Resolver`, `ExecutionContext`, `ExecutionContextMiddleware`), and the `ws` commands (`Manager`). |

The other packages import `workspace` for the model. It never imports them, so it only depends on them through its own interfaces (e.g. the git calls it needs to clone a project).

### Features

Feature packages hold the commands' handlers, and mirror the CLI's command groups.

| Package | Commands |
| ------- | -------- |
| `internal/stack` | `up`, `down`, `exec`, `ext`, `logs`, `debug` (`restart` dispatches `down` then `up`) |
| `internal/hosts` | `hosts`, with the `/etc/hosts` manager it uses |
| `internal/provision` | `provision`, with the provisioner runner |
| `internal/git` | `g`. It's also the wrapper around the git CLI that `workspace` uses to clone projects. |
| `internal/tls` | `tls gen` |
| `internal/sys` | `sys check`, `sys self-update` |
| `internal/workspace` | `ws` (see above) |

### Adapters and infrastructure

These talk to things outside of orca, or are plumbing shared by the rest.

| Package | Contents |
| ------- | -------- |
| `internal/bus` | The command bus, its middleware and hooks. It has no dependencies on the rest of orca. |
| `internal/usererr` | Attaching messages for the user to errors. |
| `internal/process` | `Executor`, which runs external commands. |
| `internal/docker` | The docker compose wrapper, and the compose overlays. |
| `internal/github` | The GitHub API client. |
| `internal/config` | The user config, `~/.orca/orca.yaml`. |
| `internal/tui` | Terminal output and prompts. |
| `internal/logging` | The slog handler. |
| `internal/plugins` | Orca's side of the plugin system, see [how plugins work](../plugins/how_it_works.md). |
| `internal/dag` | The dependency graph used to order projects. |

`pkg/` is only for public API, and only `pkg/plugin` (the plugin SDK) belongs there. Anything that's only used by orca goes in `internal/`.

## Adding a command

For example, a `ps` command that shows the running services:

1. **Define the message** in the feature package it belongs to, next to its handler. Name it `...Command`, or `...Query` if it only reads and shows something. Give it a `MessageName()`, which is the command's path joined with dots:

    ```go
    type PsQuery struct {
    	bus.Returns[bus.Void]

    	Workspace string
    	Project   string
    }

    func (PsQuery) MessageName() string {
    	return "ps"
    }
    ```

2. **Write the handler** as a method on the package's `Manager`, taking `(ctx context.Context, msg)`. If it runs against a workspace or project, read the `ExecutionContext` first:

    ```go
    func (m *Manager) Ps(ctx context.Context, query PsQuery) error {
    	execCtx, err := workspace.ExecutionContextFrom(ctx)

    	if err != nil {
    		return err
    	}

    	...
    }
    ```

3. **Register it** in `cmd/orca/services.go`. Add the handler to `GetBus()`:

    ```go
    b.HandleVoid(lazy(s.GetStack, (*stack.Manager).Ps))
    ```

    If the handler reads the `ExecutionContext`, also add a target to `GetExecutionContextMiddleware()`:

    ```go
    ec.Target(func(query stack.PsQuery) workspace.ExecutionTarget {
    	return workspace.ExecutionTarget{Workspace: query.Workspace, Project: query.Project}
    })
    ```

4. **Dispatch it** from the cobra handler, wrapped in `handleErrors`. The handler only reads flags and dispatches a message literal:

    ```go
    func handlePs(cmd *cobra.Command, args []string) error {
    	_, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.PsQuery{
    		Workspace: mustGetString(cmd, "workspace"),
    		Project:   mustGetString(cmd, "project"),
    	})

    	return err
    }
    ```

5. **Complete its arguments**, if their values come from config or the environment, such as service or branch names. See [Shell completion](#shell-completion) below. `-w` and `-p` already complete, as `addWorkspaceOption` and `addProjectOption` register them.

Two tests in `cmd/orca/services_test.go` catch wiring mistakes that would otherwise only fail at runtime:

- `Test_AllDispatchedMessagesHaveHandlers` checks every dispatched message has a handler.
- `Test_ExecutionContextHandlersHaveTargets` checks every handler that reads the `ExecutionContext` has a target.

## Shell completion

Completion runs orca as `orca __complete <args>`, on every <kbd>Tab</kbd>, so it must be fast and silent. In that mode (`completing` in `cmd/orca/completion.go`), plugins aren't started, logging is discarded, and `checkErr` prints nothing.

The values offered come from `...ChoicesQuery` messages, handled in the feature package that owns them, and returning `[]workspace.Choice`. They go through the bus like any other message, so the `ExecutionContextMiddleware` works out the workspace and project as it would for the command, from the working directory and any `-w`/`-p` typed so far. Interactive prompts can use the same queries, so the two never disagree.

```go
type ServiceChoicesQuery struct {
	bus.Returns[[]workspace.Choice]

	Workspace string
	Project   string
}

func (ServiceChoicesQuery) MessageName() string {
	return "choices.services"
}
```

Register the handler with `b.Handle(lazyResult(...))`, and give it a target if it reads the `ExecutionContext`. In `cmd/orca/completion.go`, add a completion function that dispatches the query as a literal and passes the result to `toCompletions`, which filters by what's typed, and offers nothing on an error. Use it as a command's `ValidArgsFunction` (with `firstArgOnly` if it only completes the first argument), or a flag's with `RegisterFlagCompletionFunc`.

File names are never offered: `noFileCompletions` turns them off for every builtin command that doesn't complete its own arguments.

## Dependencies

- **Packages declare the interfaces they need**, privately, next to the code that uses them (e.g. `stack` declares `compose`). Keep them to the methods that are actually called, so they're easy to fake.
- **Constructors take those interfaces**, and return the concrete type: `NewManager(compose compose) *Manager`.
- **Services are wired together in `cmd/orca/services.go`**, by getters that build each one the first time it's needed. Handlers are registered with `lazy`, so dispatching one command only builds the services it uses.
- **Don't depend on the tui for errors.** Return them instead (see [errors](./errors.md)). A service only needs the tui to show output.
- **The `ExecutionContext` is read from the `context.Context` once, in the handler**, then passed explicitly to any helpers, by convention as `execCtx`. Don't add other context values for workspace or project data.

## Conventions

- **Files** are named after what's in them. Name a package's main file after the package (`git/git.go`), never `main.go`, which is for `package main`. Use one file per command handler (`stack/up.go`, `stack/down.go`), and put the package's errors in `errors.go`.
- **Receivers** are short, and the same for every method on a type (`m *Manager`, `c *Compose`). Never `self` or `this`.
- **Feature services** are always called `Manager` (`stack.Manager`, `git.Manager`, `tls.Manager`...), built with `NewManager`. Their getters in `cmd/orca/services.go` are named after the package (`GetStack`, `GetTLS`). Avoid names that repeat the package, like `workspace.WorkspaceRepository`.
- **Options structs** that aren't messages are named `...Opts` (`git.PullBranchOpts`), not `...DTO`.
- **Errors** follow the conventions in [errors](./errors.md): return them, don't print them; use sentinels by default; check them with `errors.Is`/`errors.AsType`.
- **Exiting:** never call `os.Exit`, `cobra.CheckErr` or `log.Fatal` directly. Return an error, or use `exit`/`checkErr` from `cmd/orca/exit.go`, so plugin processes are always stopped. `Test_NoDirectExits` enforces this.

## Testing

- Tests use [testify](https://github.com/stretchr/testify), and are mostly table-driven with `t.Run`.
- Use `afero.NewMemMapFs()` for anything that touches the filesystem.
- Fake dependencies with small hand-written stubs, or mockery mocks for larger interfaces. Mocks are configured in `.mockery.yaml`, generated into `tests/mocks/<package>` with `make gen-mocks`, and committed.
- Handlers that read the `ExecutionContext` are tested by putting one in the context, rather than mocking a resolver:

    ```go
    ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws})
    ```

- Check errors with `assert.ErrorIs`/`assert.ErrorAs`, and their text with `assert.EqualError`, rather than comparing error values.
