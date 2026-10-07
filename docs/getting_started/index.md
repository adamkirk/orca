# Getting started

This page covers the ideas orca is built around. To set up a real workspace from scratch, see [Setting up a workspace](../guides/setting_up_a_workspace.md).

## Requirements

Orca shells out to a few tools, which must be on your `PATH`:

- `docker`, with the `docker compose` plugin
- `git`

Run [`orca sys check`](../CLI/orca_sys_check.md) to confirm they're available.

## Core concepts

### Projects

A **project** is a single repository (or directory) containing a docker compose file and an `orca.project.yaml`. The project config tells orca which compose file to use, which env files to pass, which hostnames and TLS certificates it needs, and so on. See the [project config reference](../configuration/project_config.md).

Each project is run as its own compose project named `orca-{workspace}-{project}`, so projects never clash with each other, or with anything you start with plain `docker compose`.

### Workspaces

A **workspace** is a named group of projects that make up an environment, e.g. an API, a frontend and the shared infrastructure they both use. It's defined by an `orca.workspace.yaml` file. That file lists each project, where to clone it from, and which other projects it depends on. See the [workspace config reference](../configuration/workspace_config.md).

The workspace file normally lives in a repository of its own, or in one of the projects (marked with `self: true`). Everyone working on the environment then shares the same definition.

### Your user config

Orca keeps track of your workspaces, and where each project is cloned on your machine, in `~/.orca/orca.yaml`. It's created on first run, and orca updates it for you when you initialise workspaces, clone projects and switch workspaces. See the [user config reference](../configuration/user_config.md).

### The current workspace

One workspace can be selected as the **current workspace** with [`orca ws switch`](../CLI/orca_ws_switch.md). It's used when you run a command outside any project directory without saying which workspace you mean. See [Choosing a workspace and project](./context.md) for exactly how that's decided.

## Quick start

Assuming someone has already written the `orca.workspace.yaml` and `orca.project.yaml` files for your environment:

```sh
# Clone the repository containing the workspace config, and go into it
$ git clone git@github.com:acme/local-env.git ~/code/acme/local-env
$ cd ~/code/acme/local-env

# Register the workspace and clone all of its projects next to this repository
$ orca ws init

# Make it the current workspace
$ orca ws switch acme

# Start everything, in dependency order. "-w" is needed because we're inside
# the local-env project, where a plain "orca up" would only start local-env.
$ orca up -w acme
```

Then, from inside any project's directory, commands apply to just that project:

```sh
$ cd ~/code/acme/api
$ orca logs              # follow this project's logs
$ orca exec -s app sh    # open a shell in its "app" service
$ orca restart           # restart just this project
```

## Where next

- [Choosing a workspace and project](./context.md): how every command decides what to act on.
- [Setting up a workspace](../guides/setting_up_a_workspace.md): a full worked example, from empty repositories to a running environment.
- [Using orca day to day](../usage/index.md): every feature, by topic.
