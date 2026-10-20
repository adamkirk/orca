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
make lint-last-commit  # commitlint the last commit, as CI does for every commit in a PR
```

Before finishing a change, run `go build ./...`, `make vet`, `./scripts/fmt_check.sh` and `make test`. Go 1.27 or later is required, because the bus uses generic methods.

## Layout

```
cmd/orca/            the only package that knows about cobra, flags and wiring
  services.go        builds services lazily; GetBus() registers every handler,
                     GetExecutionContextMiddleware() registers every execution target
  exit.go            handleErrors/reportError (the one place errors are shown), exit/checkErr
  completion.go      shell completion: completing mode, and the functions that dispatch ...ChoicesQuery
internal/
  workspace/         domain: model, config loading (Repository), Resolver, ExecutionContext
                     and its middleware, and the `ws` commands (Manager)
  stack/ hosts/ git/ tls/ sys/
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
2. **Handler:** a method on the package's `Manager`, `func (m *Manager) X(ctx context.Context, cmd XCommand) error`. It writes output with the tui and returns errors. It never prints them.
3. **Register** in `cmd/orca/services.go`, in `GetBus()`: `b.HandleVoid(lazy(s.GetStack, (*stack.Manager).X))`.
4. **Execution context:** if the command runs against a workspace or project:
   - The handler reads it first, with `execCtx, err := workspace.ExecutionContextFrom(ctx)`.
   - Register a target in `GetExecutionContextMiddleware()`. Set `WorkspaceOptional` if the command still works outside a workspace, as the git commands do.
   - Only call `ExecutionContextFrom` in the handler itself, and pass `execCtx` explicitly to helpers. Don't add other context values for workspace or project data.
5. **CLI:** the cobra handler, wrapped in `handleErrors`, only reads flags and dispatches a message *literal*: `_, err := svcContainer.GetBus().Dispatch(cmd.Context(), stack.XCommand{...})`.
6. **Completion:** every argument or flag whose value comes from config or the environment (service, extension, branch names...) completes:
   - Add an `XChoicesQuery` returning `[]workspace.Choice` (`bus.Returns[[]workspace.Choice]`, `MessageName` `"choices.x"`) in the feature package. Register it with `b.Handle(lazyResult(...))`, plus a target if it reads the execution context.
   - In `cmd/orca/completion.go`, add a `completeX` that dispatches it as a literal and returns `toCompletions(choices, err, toComplete)`. Use it as the command's `ValidArgsFunction` (`firstArgOnly(...)` for the first argument only), or with `RegisterFlagCompletionFunc`.
   - `-w` and `-p` already complete. File names are never offered, `noFileCompletions` turns them off for any command without its own completion.
   - Completion runs on every <tab>, so it must be fast and silent: never start plugins, print, prompt or log. Add the new values to "What completes" in `docs/usage/shell_completion.md`.

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
- **Receivers:** short and consistent per type (`m *Manager`, `c *Compose`). Never `self`/`this`.
- **Names:**
  - Don't repeat the package in names (`workspace.Repository`, not `workspace.WorkspaceRepository`).
  - Every feature package's service is called `Manager` (`stack.Manager`, `git.Manager`...), built with `NewManager`, and has a `GetX()` getter in `services.go` named after the package (`GetStack`, `GetTLS`).
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

## Issues and planning

Work is tracked in GitHub issues on `adamkirk/orca`, and planned in the [Orca roadmap](https://github.com/users/adamkirk/projects/2) project. Use the `gh` CLI for both. Project commands need the `project` scope: `gh auth refresh -s project,read:project`.

- **The issue is the spec.** A feature issue describes the behaviour from the user's point of view. Its acceptance criteria are the definition of done. Read it with `gh issue view <n>` before starting, and check every criterion before calling the work done. There are no local copies of the specs. Don't create any, e.g. in `docs/proposals/`.
- **Feature issues share a structure:**
  - Summary
  - Who it's for
  - User stories
  - Behaviour
  - Configuration
  - Rules and edge cases
  - Interactions
  - Out of scope
  - Open questions, each with a recommended answer
  - Acceptance criteria

  Follow it for new ones, and refer to other issues by number (`#49`).
- **Keep the spec current.** When a decision changes, or an open question is answered, update the issue body with `gh issue edit <n> --body-file <file>`. A settled answer moves out of "Open questions" and into the rules. Ask open questions one at a time, with context and a recommended answer, rather than guessing.
- **Order and dependencies:**
  - The project's Order field, and its "Implementation order" view, give the sequence to work in.
  - Dependencies are GitHub "blocked by" relationships. Check them before starting:

    ```sh
    gh project item-list 2 --owner adamkirk --format json \
      | jq -r '.items | sort_by(.order) | .[] | "\(.order)  #\(.content.number)  \(.status)  \(.content.title)"'
    gh api repos/adamkirk/orca/issues/<n>/dependencies/blocked_by -q '.[].number'
    ```

  - New items go to the bottom of the project, so set their Order, and position them, when adding them.
- **Labels:** every issue gets one type label (`enhancement`, `bug`, `documentation`, `tests`, `coding standards`, `release-process`), plus any that apply:
  - `UX`, for user-facing changes
  - one or more scope labels: `scope:workspace`, `scope:cli`, `scope:plugins`, `scope:git`, `scope:tls`

  Use the existing labels. Only add a new `scope:` label when no existing one fits.
- **Shipping:** a PR that implements an issue says `Closes #<n>`, and its commits follow the conventions below. Issues aren't user docs, so document the shipped behaviour in `docs/` as part of the same change.
- **Only change GitHub when asked.** Creating, editing or closing issues, labels or project items is visible to everyone, so don't do it unprompted. Suggest the change instead.

## Docs and commits

- The docs use mkdocs:
  - Add new pages to `nav` in `mkdocs.yml`.
  - `docs/CLI` is generated by `orca util gen-docs` and not committed, so command and flag help belongs in the cobra definitions.
- Update `docs/development_guide/` and this file when changing how the code is structured.
- Commits follow [conventional commits](https://www.conventionalcommits.org) (`feat:`, `fix:`, `refactor:`, `docs:`, `chore:`), checked by commitlint in CI. PRs are squash-merged.
- **Lint every commit.** CI fails the PR if any of its commits fails commitlint, so run `make lint-last-commit` straight after each commit, and fix it with `git commit --amend` before moving on. Fixing one later means rewriting history, and force-pushing if it's been pushed.
  - Only use issue references (`#51`) in a footer line of their own, like `Part of #51.` or `Closes #51`, after a blank line. One in the middle of the body, like `(#50)`, is parsed as the start of the footer, and fails `footer-leading-blank`.
- **Never mark a commit as breaking**, with `!` (`feat!:`) or a `BREAKING CHANGE:` footer, even for changes that are. Orca is pre-1.0, so any release can change behaviour, as Terraform did before 1.0. A breaking commit would make semantic-release publish `1.0.0`, and the maintainer decides when that happens. Instead:
  - Use the normal type.
  - Describe the change in behaviour, and what users should do about it, in the commit body.
  - Document it in `docs/`.

  See [release_process.md](docs/development_guide/release_process.md#versioning-before-10).
