# Orca

Orca is a Go CLI that runs development environments spread across several repositories, as one unit. Each repository keeps its own `docker-compose.yml`, and orca starts, stops and connects them, with helpers for git across all of them.

`AGENTS.md` is a symlink to this file. The full developer docs are in `docs/development_guide/`. This file summarises the rules from them, so keep the two in step:

- [architecture.md](docs/development_guide/architecture.md): package layout, how a command runs, adding a command
- [command_bus.md](docs/development_guide/command_bus.md): messages, handlers, middleware, hooks, execution context
- [errors.md](docs/development_guide/errors.md): error handling

## Commands

```sh
make build          # gofmt, then build to .local/bin/orca
make test           # go test with coverage; TESTS=./internal/bus/... to narrow it
make vet            # go vet ./...
make fmt            # gofmt -w; CI runs ./scripts/fmt_check.sh and fails on any diff
make gen-mocks      # regenerate tests/mocks from .mockery.yaml (commit the result)
make gen-proto      # regenerate pkg/plugin/proto/v1 from proto/ (commit the result)
```

Before finishing a change, run `go build ./...`, `make vet`, `./scripts/fmt_check.sh` and `make test`. Go 1.27 or later is required, because the bus uses generic methods.

## Layout

```
cmd/orca/            the only package that knows about cobra, flags and wiring
  services.go        builds services lazily; GetBus() registers every handler,
                     GetExecutionContextMiddleware() registers every execution target
  exit.go            handleErrors/reportError (the one place errors are shown), exit/checkErr
internal/
  workspace/         domain: model, config loading (Repository), Resolver, ExecutionContext
                     and its middleware, and the `ws` commands (Manager)
  stack/ hosts/ provision/ git/ tls/ sys/
                     features: command handlers, one package per CLI command group
  bus/ usererr/ process/ docker/ github/ config/ tui/ logging/ plugins/ dag/
                     adapters and infrastructure
pkg/plugin/          the public plugin SDK, the only package in pkg/
```

- Don't create `common`, `util`, `helpers`, `controller` or `repository` packages, or anything else named after a layer. Put code in the package that owns the concept.
- `internal/workspace` must not import the feature packages. It depends on them through its own interfaces.
- `pkg/` is public API. Only add to it if external code should import it. Treat changes to `pkg/plugin` as breaking unless they're additive, and bump the handshake's `ProtocolVersion` for breaking protocol changes.

## Adding or changing a command

Every command that does something goes through the bus. The exceptions are ones that only inspect orca itself: `version`, `config`, `plugins ls` and `util gen-docs`.

1. **Message:** in the feature package, next to its handler. Name it `XCommand`, or `XQuery` if it only reads and shows something. Embed `bus.Returns[bus.Void]`, and add `MessageName()`, returning the command's path joined with dots (`"ws.init"`, `"g.co"`).
2. **Handler:** a method on the package's service, `func (m *Manager) X(ctx context.Context, cmd XCommand) error`. It writes output with the tui and returns errors. It never prints them.
3. **Register** in `cmd/orca/services.go`, in `GetBus()`: `b.HandleVoid(lazy(s.GetStack, (*stack.Manager).X))`.
4. **Execution context:** if the command runs against a workspace or project:
   - The handler reads it first, with `execCtx, err := workspace.ExecutionContextFrom(ctx)`.
   - Register a target in `GetExecutionContextMiddleware()`. Set `WorkspaceOptional` if the command still works outside a workspace, as the git commands do.
   - Only call `ExecutionContextFrom` in the handler itself, and pass `execCtx` explicitly to helpers. Don't add other context values for workspace or project data.
5. **CLI:** the cobra handler, wrapped in `handleErrors`, only reads flags and dispatches a message *literal*: `_, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.XCommand{...})`.

`cmd/orca/services_test.go` checks the wiring, and fails if a dispatched message has no handler, or a handler reading the execution context has no target. That only works if messages are dispatched as literals, and `ExecutionContextFrom` is only called in handlers.

Bus specifics:

- `Bus` is a concrete type with generic methods (`Handle`, `HandleVoid`, `Dispatch`, `Before`, `After`). Interfaces can't declare generic methods, so don't try to put the bus behind an interface. Fake handlers in tests instead.
- Middleware (`b.Use`) wraps hooks and handlers, and can replace the `context.Context`. Hooks only observe or veto: a `Before` hook's error stops the command.

## Errors

- **Return errors, never print them.** `reportError` in `cmd/orca/exit.go` shows each failure once. Don't add tui calls to report errors.
- **Messages for the user:** `return usererr.Wrap(err, "Failed to do X!")`. It returns nil for a nil error, so `return usererr.Wrap(doX(), "...")` is fine. Only the innermost message in a chain is shown, followed by its cause.
- **Context:** wrap with `%w`: `fmt.Errorf("loading %s: %w", name, err)`. Wrap underlying errors, rather than flattening them with `err.Error()`.
- **Checking:** use `errors.Is` and `errors.AsType[T]`. Never use `==` or type assertions on errors.
- **Defining** errors in the owning package's `errors.go`:
  - Sentinels by default: `var ErrUnknownProject = errors.New("unknown project")`, returned as `fmt.Errorf("%w: %s", ErrUnknownProject, name)`.
  - A struct type only if callers read its fields. Name it `XxxError` (`InvalidWorkspaceConfigError`), with a value receiver, and `Unwrap` if it wraps another error.
  - Use `Err` for values and `...Error` for types, as the standard library does.

## Conventions

- **Files:** name a package's main file after the package (`git/git.go`), never `main.go`. Use one file per command handler, and put the package's errors in `errors.go`.
- **Receivers:** short and consistent per type (`m *Manager`, `g *Git`). Never `self`/`this`.
- **Names:**
  - Don't repeat the package in names (`workspace.Repository`, not `workspace.WorkspaceRepository`).
  - New feature services are `Manager`.
  - Options structs that aren't messages are `...Opts`, never `...DTO`.
- **Dependencies:**
  - Declare small private interfaces in the package that uses them, with only the methods it calls.
  - Constructors take interfaces and return concrete types.
  - A service only depends on the tui if it shows output.
- **Exiting:** never call `os.Exit`, `cobra.CheckErr` or `log.Fatal`. Return an error, or use `exit`/`checkErr` in `cmd/orca/exit.go`, so plugin processes are stopped. `Test_NoDirectExits` enforces this.
- **Utilities:** prefer the standard library (`slices`, `maps`, `errors`) over writing helpers.

## Tests

- testify, table-driven with `t.Run`. Use `afero.NewMemMapFs()` for the filesystem.
- Fake dependencies with small stubs, or mockery mocks listed in `.mockery.yaml`. After changing a mocked interface, run `make gen-mocks`.
- To test a handler that reads the execution context, put one in the context:
  `workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws})`.
- Check errors with `assert.ErrorIs`/`ErrorAs`, and their text with `EqualError`, rather than comparing error values.

## Docs and commits

- The docs use mkdocs:
  - Add new pages to `nav` in `mkdocs.yml`.
  - `docs/CLI` is generated by `orca util gen-docs` and not committed, so command and flag help belongs in the cobra definitions.
- Update `docs/development_guide/` and this file when changing how the code is structured.
- Commits follow [conventional commits](https://www.conventionalcommits.org) (`feat:`, `fix:`, `refactor:`, `docs:`, `chore:`), checked by commitlint in CI. PRs are squash-merged.
