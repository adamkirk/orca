# Plugins

Plugins add your own commands to orca, without changing orca itself. A plugin is a separate program that orca starts and talks to over gRPC, so a broken plugin can't crash orca. This is the same approach Terraform uses for its providers.

Use them for team-specific workflows that don't belong in orca itself: seeding test data, talking to internal services, wrapping your own tooling.

- To install and use plugins, read on.
- To write one, see [Building a plugin](./building.md).
- For the protocol and architecture, see [How plugins work](./how_it_works.md).

## Installing a plugin

A plugin is a single executable whose name starts with `orca-plugin-`, e.g. `orca-plugin-seed`. To install one, put it in orca's plugins directory and make sure it's executable:

```sh
$ mkdir -p ~/.orca/plugins
$ cp orca-plugin-seed ~/.orca/plugins/
$ chmod +x ~/.orca/plugins/orca-plugin-seed
```

Its commands then show up alongside orca's own:

```sh
$ orca --help
...
Available Commands:
  ...
  seed        Seeds the database with test data.
  ...
```

To see which plugins are loaded, and the commands each provides, run [`orca plugins ls`](../CLI/orca_plugins_ls.md):

```sh
$ orca plugins ls
seed (/Users/me/.orca/plugins/orca-plugin-seed)
  commands: seed
```

## Plugin directories

Orca looks for plugins in:

1. `~/.orca/plugins`, or the directory in `ORCA_PLUGINS_PATH` if it's set.
2. Each directory in `plugins.dirs` in your [user config](../configuration/user_config.md), in order.

```yaml
plugins:
  dirs:
    - /Users/me/code/acme/orca-plugins/bin
```

Extra directories are useful for keeping a team's plugins in a shared repository. If two directories contain a plugin with the same file name, the first one found is used.

Only regular files that are executable and named `orca-plugin-*` are loaded. Anything else in these directories is ignored.

## When things go wrong

Orca carries on without a plugin if it can't load it, e.g. because it crashed or isn't really a plugin, and logs a warning. Orca's own commands always win: if a plugin adds a command with the same name as a built-in one (or another plugin's), it's skipped with a warning. Run with `--log-level warn` (or `debug`) to see these:

```sh
$ orca --log-level warn plugins ls
```

## Things to be aware of

- **Plugins start on every run.** Orca starts every plugin each time it runs, to find out which commands they provide. Each one adds a few milliseconds, so uninstall plugins you don't use.
- **Plugins run as you.** A plugin can do anything you can, so only install plugins you trust.
- **No interactive input.** Plugin commands can print output, but can't read from your terminal yet.
- **Plugins are stopped with orca.** When orca exits, or is interrupted with Ctrl-C, it stops its plugins. Plugins built with the Go SDK also exit by themselves if orca is killed without warning.
