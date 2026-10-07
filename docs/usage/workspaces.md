# Workspaces

The `orca ws` commands register workspaces on your machine, clone their projects, and choose which workspace is current. For a full example, see [Setting up a workspace](../guides/setting_up_a_workspace.md).

## Initialising a workspace

```sh
$ cd ~/code/acme/local-env      # a repository containing orca.workspace.yaml
$ orca ws init
```

[`orca ws init`](../CLI/orca_ws_init.md):

1. Reads the workspace config from the current directory, and registers the workspace in your [user config](../configuration/user_config.md).
2. Sets up every project in it, as [`orca ws clone`](#cloning-projects) does.

| Flag | Description | Default |
| ---- | ----------- | ------- |
| `-s`, `--source` | The directory containing the workspace config. | The current directory |
| `-c`, `--config` | The workspace config's file name. | `orca.workspace.yaml` |
| `-t`, `--target` | The directory to clone projects into. | The parent of the current directory |

It doesn't make the workspace current; use `orca ws switch` for that.

## Cloning projects

```sh
$ orca ws clone                 # every project in the workspace
$ orca ws clone -p payments     # just one project
```

[`orca ws clone`](../CLI/orca_ws_clone.md) sets up any projects that aren't set up yet. Run it after a project is added to the workspace config. For each project:

- A `self: true` project is registered at the root of the workspace config's repository.
- If its directory exists and the project is already registered, it's skipped.
- If its directory exists and is a clone of the project's repository (checked by its git remote), it's registered as is.
- If its directory exists but is something else, orca stops with an error, rather than overwrite it.
- Otherwise the repository is cloned and registered.

Projects are cloned into `{target}/{project name}`. `--target` defaults to the parent of the workspace config's repository, so projects end up next to it. With `-p` and `--target`, the project is cloned into `--target` itself.

## Switching workspaces

| Command | Does |
| ------- | ---- |
| [`orca ws ls`](../CLI/orca_ws_ls.md) | Lists your workspaces, and which is current. |
| [`orca ws switch <name>`](../CLI/orca_ws_switch.md) | Makes a workspace the current one. |
| [`orca ws current`](../CLI/orca_ws_current.md) | Shows the current workspace. |
| [`orca ws clear-current`](../CLI/orca_ws_clear-current.md) | Unsets the current workspace. |

The current workspace is used when you're not inside any project's directory and don't pass `-w`. See [Choosing a workspace and project](../getting_started/context.md).
