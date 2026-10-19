# Errors

## Return errors, don't print them

Code that fails returns an error, and never prints it. Errors are shown to the user in one place, `reportError` in `cmd/orca/exit.go`, once they've made it back to the command. That way every failure is shown exactly once, and none are lost.

## Messages for the user

To give the user a friendlier message than the error itself, wrap it with `usererr.Wrap`:

```go
if err := cm.createTLSDirsIfMissing(); err != nil {
	return usererr.Wrap(err, "Failed to create certificates directory!")
}
```

`Wrap` returns `nil` for a `nil` error, so it can wrap the result of a call directly:

```go
return usererr.Wrap(g.checkoutOrCreate(dir, name), "Failed to checkout branch")
```

When the error reaches the command, the user sees the message, followed by the error it wraps:

```
Failed to create certificates directory!
  mkdir /home/me/.orca/tls: permission denied
```

If there's more than one message in the chain, only the innermost one is shown, as it's the most specific. Errors without a message are shown as `Error: <error>`.

Errors the user caused by aborting a prompt (`tui.ErrUserAbortedExecution`) aren't shown at all. Problems with a workspace config (`workspace.InvalidWorkspaceConfigError`) are listed in full.

## Adding context

Wrap errors with `%w` to add context as they're passed up, so the chain can still be checked:

```go
return fmt.Errorf("loading the compose spec for '%s': %w", p.Name, err)
```

## Checking errors

Always use `errors.Is` and `errors.AsType` (or `errors.As`), never compare errors or use type assertions. Those stop matching as soon as an error is wrapped.

```go
if errors.Is(err, tui.ErrUserAbortedExecution) { ... }

if invalidCfg, ok := errors.AsType[workspace.InvalidWorkspaceConfigError](err); ok { ... }
```

## Defining errors

Errors live in the package that returns them, in its `errors.go`. Follow the standard library's naming: error values start with `Err` (`io.EOF`, `fs.ErrNotExist`), and error types end in `Error` (`os.PathError`).

Use a sentinel value by default, and wrap it with any detail where it's returned. Callers can still check for it with `errors.Is`:

```go
var ErrUnknownProject = errors.New("unknown project")

return fmt.Errorf("%w: %s", ErrUnknownProject, name) // "unknown project: api"
```

Only define a type when callers need data from the error, and name it `...Error`:

```go
type InvalidWorkspaceConfigError struct {
	Path     string
	Problems []string
}
```

Use value receivers for its `Error` method, as the existing types do, so callers match it with `errors.AsType[T]` rather than `errors.AsType[*T]`. If it wraps another error, give it an `Unwrap` method too (see `bus.HookFailedError`).
