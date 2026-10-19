# Workspaces

The `orca ws` commands register workspaces on your machine, clone their projects, and choose which workspace is current. For a full example, see [Setting up a workspace](../guides/setting_up_a_workspace.md).

## Initialising a workspace

You can initialise a workspace straight from its GitHub repository, or from a copy of the repository you've already cloned.

### From GitHub

```sh
$ mkdir -p ~/code/acme && cd ~/code/acme
$ orca ws init --repo acme/local-env
```

With `--repo`, [`orca ws init`](../CLI/orca_ws_init.md):

1. Fetches the workspace config from the repository's default branch, or `--branch`, through the GitHub API, and checks that it's [valid](../configuration/workspace_config.md#validation). If the branch or the config is missing, or the config is invalid, orca stops before cloning anything.
2. Checks that every project's [`repository.branch`](../configuration/workspace_config.md#repository) exists on its remote. If any are missing, orca lists them all and stops before cloning anything.
3. Clones the repository over SSH (`git@github.com:acme/local-env.git`) into `{target}/{repo name}`, here `~/code/acme/local-env`, on `--branch` if it's given. If that directory already exists, orca stops with an error rather than touch it.
4. Carries on as if you'd run `orca ws init` from inside the clone (see [below](#from-a-local-directory)), so the other projects are cloned next to it.

`--repo` only accepts GitHub repositories, in the form `owner/repo`. You'll need SSH access to the repository for the clone.

For a private repository, set `GH_TOKEN` or `GITHUB_TOKEN` to a token that can read it, as you would for the [GitHub CLI](https://cli.github.com/manual/gh_help_environment). `GH_TOKEN` is used if both are set. Without a token, a private repository looks the same as one without a workspace config.

### From a local directory

```sh
$ cd ~/code/acme/local-env      # a repository containing orca.workspace.yaml
$ orca ws init
```

Without `--repo`, [`orca ws init`](../CLI/orca_ws_init.md):

1. Reads the workspace config from the current directory, checks that it's [valid](../configuration/workspace_config.md#validation), and registers the workspace in your [user config](../configuration/user_config.md).
2. Sets up every project in it, as [`orca ws clone`](#cloning-projects) does.

### Flags

| Flag | Description | Default |
| ---- | ----------- | ------- |
| `-r`, `--repo` | The GitHub repository to initialise from, as `owner/repo`. Can't be used with `--source`. | |
| `-b`, `--branch` | The branch of the `--repo` repository to read the workspace config from, and to clone. It only applies to the workspace repository: each project's branch comes from its `repository.branch`. Can't be used with `--source`, which uses the local checkout on whichever branch it's on. | The repository's default branch |
| `-s`, `--source` | The directory containing the workspace config. | The current directory |
| `-c`, `--config` | The workspace config's file name, relative to the source directory or the repository's root. | `orca.workspace.yaml` |
| `-t`, `--target` | The directory to clone projects into. | The current directory with `--repo`, otherwise its parent |

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
- Otherwise the repository is cloned, on its `repository.branch` if it has one, and registered.

Before cloning anything, orca checks that the `repository.branch` of every project it's about to clone exists on its remote, and lists every missing one. The branch is only used for the first clone: projects that are already cloned are never switched to another branch.

Projects are cloned into `{target}/{project name}`. `--target` defaults to the parent of the workspace config's repository, so projects end up next to it. With `-p` and `--target`, the project is cloned into `--target` itself.

## Switching workspaces

| Command | Does |
| ------- | ---- |
| [`orca ws ls`](../CLI/orca_ws_ls.md) | Lists your workspaces, and which is current. |
| [`orca ws switch <name>`](../CLI/orca_ws_switch.md) | Makes a workspace the current one. |
| [`orca ws current`](../CLI/orca_ws_current.md) | Shows the current workspace. |
| [`orca ws clear-current`](../CLI/orca_ws_clear-current.md) | Unsets the current workspace. |

The current workspace is used when you're not inside any project's directory and don't pass `-w`. See [Choosing a workspace and project](../getting_started/context.md).
