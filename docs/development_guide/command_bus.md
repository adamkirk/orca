# Command bus

Commands are dispatched through a command bus (`internal/bus`) rather than the CLI handlers calling services directly. This gives a single place for behaviour that applies to every command, through middleware and hooks.

The exceptions are commands that only inspect orca itself, rather than doing anything, such as `version`, `config`, `plugins ls` and `util gen-docs`, and commands provided by plugins.

## Messages

Each command or query is a struct that embeds `bus.Returns[R]` to declare its result type, and has a `MessageName()` that identifies it. By convention it's the command's path, joined with dots (e.g. `ws.init`). Messages that only read and show something are named `...Query`, and the rest `...Command`. For now, both write their output to the tui rather than returning it, so they use `bus.Void`.

```go
type UpCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Project   string
}

func (UpCommand) MessageName() string {
	return "up"
}
```

## Handlers

Handlers are service methods, which take the message and return its result (or just an error, for `bus.Void`):

```go
func (c *Controller) Up(ctx context.Context, cmd UpCommand) error
```

They're registered in `GetBus()` in `cmd/orca/services.go`. `lazy` wraps the method so the service is only built when its message is dispatched, rather than every service being built with the bus. `HandleVoid` is for messages that return `bus.Void`, and `Handle` for everything else.

```go
b.HandleVoid(lazy(s.GetController, (*controller.Controller).Up))
```

A message without a handler only fails when it's dispatched, so `Test_AllDispatchedMessagesHaveHandlers` checks that every message the commands dispatch has one. For that to work, messages must be dispatched as literals.

CLI handlers then dispatch the message. The result type is inferred from the message, so no type arguments are needed:

```go
_, err := svcContainer.GetBus().Dispatch(cmd.Context(), controller.UpCommand{...})
```

The bus uses generic methods, so needs Go 1.27 or later. Interfaces can't declare generic methods, so the bus is a concrete type, and tests should fake the handlers rather than the bus.

## Middleware

Middleware wraps every dispatch, including its hooks, and runs in the order it was added with `Use`. `bus.LoggingMiddleware` logs each dispatch at debug level.

## Hooks

Hooks run before and after a message is handled:

- `Before` hooks run before the handler. If one returns an error, the dispatch stops there.
- `After` hooks always run, with the handler's result and error. Their errors are joined with the handler's.

`b.Before(...)` and `b.After(...)` add hooks for a single message type, and `b.AddHook(...)` adds one that sees every message as a `bus.Event`.

## Execution context

Most commands run against a workspace and/or project, which is worked out from the flags, the current directory, or the current workspace. Rather than each handler doing this itself, `workspace.ExecutionContextMiddleware` resolves the `ExecutionContext` before the command is handled, and passes it along in the `context.Context`:

```go
func (c *Controller) Up(ctx context.Context, cmd UpCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	...
}
```

It's only resolved for commands with a target registered in `GetExecutionContextMiddleware()`, in `cmd/orca/services.go`, which says where the workspace and project come from:

```go
ec.Target(func(cmd controller.UpCommand) workspace.ExecutionTarget {
	return workspace.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project}
})
```

Set `WorkspaceOptional` for commands that still work without a workspace, such as the git commands, which then get an `ExecutionContext` with only the working directory.

If it can't be resolved, the middleware shows the error and the command isn't run. Hooks run after the middleware, so they can read the `ExecutionContext` too.

`Test_ExecutionContextHandlersHaveTargets` checks that every handler calling `workspace.ExecutionContextFrom` has a target registered for its message, so it should only be called from the handler itself, and helpers should take the `ExecutionContext` as an argument.

The `context.Context` only carries the `ExecutionContext` from the middleware to the handler. After that it's passed explicitly, by convention as `execCtx`, so helpers say what they depend on and the compiler can check it. Don't add other context values for workspace or project data.
