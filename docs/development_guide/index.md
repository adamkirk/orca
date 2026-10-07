# Development guide

This repository uses [Github Flow](https://docs.github.com/en/get-started/using-github/github-flow) branching strategy; feature branches come off of main, should be short-lived, and then are merged back into main. The configuration of the repository is configured to allow only squash merging, to keep the commit history clean and informative. To aid this it is following the [conventional commit standard](https://www.conventionalcommits.org/en/v1.0.0/), which is enforced via a CI check.

## Commit standards

As mentioned above we use conventional commits. In order to ensure this there is a CI workflow that runs on each PR, whichc checks that every commit on the head branch (that isn't on the base) follows this convention. To do this we use [commitlintjs](https://commitlint.js.org/), and you'll find the exact configuration for this check in `commitlint.config.js`.

### Running the commit lint

You can easily run the commit lint script locally for the latest commit with `npx commitlint --latest` from the root of the repository.

## Documentation

The documentation is built with [mkdocs](https://www.mkdocs.org/), and more specifically we use [material](https://squidfunk.github.io/mkdocs-material/) to add few nice features. There are 2 key parts to this, the `mkdocs.yml` config in the root of this repository, and the `docs` folder. The `mkdocs.yml` config defins things like plugins and directories that mkdocs uses to build static html, and the `docs` folder contains all the source markdown files.

The usage of some of the features in mkdocs will result in the docs not rendering particularly well inside github itself, but the github pages site for this repository should present it all nicely. 

The site's navigation is defined by `nav` in `mkdocs.yml`, so add any new page there too, or it won't appear in the menu.

### Working locally

If you're writing docs locally and wanna see what the rendered result looks like you can use the mkdocs server in watch mode. This is all hooked up in the `docker-compose.yml` file. Simply run `make dc-up` in the root of this repository, and you'll be able to see the rendered docs on [http://localhost:9898](http://localhost:9898). Every time either the `mkdocs.yml` config or a file in the `docs` directory changes, it will be re-rendered and any browsers will reload the page.

### CLI reference

The pages in `docs/CLI` are generated from the cobra commands, so they're always in sync with the CLI. They're not committed (the directory is gitignored). The `deploy_github_pages.yaml` workflow builds orca and runs `orca util gen-docs` before building the site, and it redeploys whenever `cmd/orca` changes, as well as `docs`.

To generate them locally, so links to them work while you're previewing:

```sh
$ make build
$ ORCA_CONFIG_PATH="$(mktemp -d)/orca.yaml" ORCA_PLUGINS_PATH=/nonexistent \
    ./.local/bin/orca util gen-docs
```

The two environment variables give orca a fresh config and no plugins, as in CI. Without them, any plugins you have installed would add their commands to the pages, and flag defaults that come from your config (like `--log-level`) would show your settings.

Because the pages come from the CLI, command and flag descriptions belong in the cobra command definitions (each command's `Short`, `Long` and flag usage strings). The hand-written pages explain concepts and workflows, and link to the CLI pages for the details.

## Plugins

The plugin system is described in [How plugins work](../plugins/how_it_works.md). A few things to know when working on it:

- **The gRPC code is generated** from `proto/orca/plugin/v1/plugin.proto` into `pkg/plugin/proto/v1`, and committed. Run `make gen-proto` after changing the proto file. It uses `buf`, `protoc-gen-go` and `protoc-gen-go-grpc`, which are pinned as `tool` dependencies in `go.mod`, so nothing needs installing.
- **The example plugin** in `examples/plugins/hello` is built into your plugins directory with `make build-example-plugins`.
- **`pkg/plugin` is a public API.** External plugins import it, so treat changes to it as breaking unless they're additive. Bump the handshake's `ProtocolVersion` for breaking protocol changes.
- **Never exit directly.** Use `exit`/`checkErr` from `cmd/orca/exit.go`, or return an error, so plugin processes are always stopped. `Test_NoDirectExits` fails the build on any direct `os.Exit`, `cobra.CheckErr` or `log.Fatal` in `cmd/orca` or `internal/`.

## Requirements

The only requirements for developing this tool are shown below (linked to the installation instructions):

- [golang >= 1.27](https://go.dev/doc/install)

Other tools used for code generation are pinned as `tool` dependencies in `go.mod`, and run with `go tool`, so they don't need installing:

| Tool | Used by |
| ---- | ------- |
| [mockery](https://vektra.github.io/mockery/latest/) | `make gen-mocks`, which generates the mocks in `tests/mocks` from `.mockery.yaml`. |
| [buf](https://buf.build/docs/), `protoc-gen-go`, `protoc-gen-go-grpc` | `make gen-proto`, which generates the plugin gRPC code. |

To upgrade one, run e.g. `go get -tool github.com/vektra/mockery/v3@latest`, then regenerate.

## Why npm?

NPM isn't actually used by the app in any way, but a couple of the tools we're using require it. We need to install some plugins for [semantic-release](https://semantic-release.gitbook.io/semantic-release/) and [commitlintjs](https://commitlint.js.org/), so we have an npm file structure to enable these tools.