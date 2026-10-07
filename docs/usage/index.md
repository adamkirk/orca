# Using orca

These pages cover what you can do with orca, by topic. Every command also has a page in the [CLI reference](../CLI/orca.md) listing all of its flags.

| Topic | Commands |
| ----- | -------- |
| [Environments](./environments.md) | `up`, `down`, `restart`, `logs`, `exec`, `debug` |
| [Extensions & provisioners](./extensions_and_provisioners.md) | `ext`, `provision` |
| [Hosts & TLS](./hosts_and_tls.md) | `hosts`, `tls gen` |
| [Git](./git.md) | `g co`, `g branches`, `g status`, `g pull`, `g push`, `g logl`, `g rbi`, `g undo` |
| [Workspaces](./workspaces.md) | `ws init`, `ws clone`, `ws ls`, `ws switch`, `ws current`, `ws clear-current` |
| [Plugins](../plugins/index.md) | `plugins ls`, and any commands your plugins add |
| [Orca itself](./system.md) | `sys check`, `sys install`, `sys self-update`, `version`, `config`, `completion` |

Most commands work out which workspace and project to act on from your current directory. See [Choosing a workspace and project](../getting_started/context.md).

## Global flags

These work with every command:

| Flag | Description |
| ---- | ----------- |
| `--log-level` | `debug`, `info`, `warn`, `error` or `none`. Overrides `logging.level` in your [user config](../configuration/user_config.md). |
| `--log-format` | `text`, or anything else for JSON. |
| `-h`, `--help` | Help for the command. |
