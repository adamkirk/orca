# Choosing a workspace and project

Most commands act on either a single project, or every project in a workspace. You can say which explicitly with `-w/--workspace` and `-p/--project`, but usually orca works it out from where you run the command.

## How it's decided

| `-w` given | `-p` given | Result |
| :--------: | :--------: | ------ |
| Yes | Yes | That project, in that workspace. |
| Yes | No | Every project in that workspace. |
| No | Yes | That project, in the workspace of the project you're currently in. If you're not in a project directory, the current workspace is used instead. |
| No | No | If you're inside a project's directory, just that project. Otherwise, every project in the current workspace. |

"Inside a project's directory" means your working directory is the project's directory, or anywhere below it. The project directories are the paths recorded in your [user config](../configuration/user_config.md). If projects are nested inside each other, the deepest one containing your directory is used.

If orca needs the current workspace and none is selected, the command fails. Either select one with [`orca ws switch`](../CLI/orca_ws_switch.md), or pass `-w`.

## Examples

Given a workspace `acme` with projects `local-env`, `api` and `web`, where `acme` is the current workspace:

```sh
$ cd ~/code/acme/api
$ orca up                     # starts just "api"
$ orca up -p web              # starts just "web" (in acme, because we're in an acme project)
$ orca up -w acme             # starts every project in acme

$ cd ~
$ orca up                     # starts every project in the current workspace (acme)
$ orca up -w other -p db      # starts "db" in the "other" workspace
```

## Commands that need a single project

Some commands only make sense for one project. They fail if the context resolves to a whole workspace:

- [`orca exec`](../CLI/orca_exec.md)
- [`orca logs`](../CLI/orca_logs.md)
- [`orca ext`](../CLI/orca_ext.md)
- [`orca provision`](../CLI/orca_provision.md)
- [`orca debug show-compose-command`](../CLI/orca_debug_show-compose-command.md) and [`show-compose-config`](../CLI/orca_debug_show-compose-config.md)

Run them from inside the project's directory, or pass `-p`.

!!! note
    `orca provision` doesn't accept `-w` or `-p` yet, so it must be run from inside the project's directory.

## Commands that always use the whole workspace

[`orca hosts`](../CLI/orca_hosts.md), [`orca tls gen`](../CLI/orca_tls_gen.md) and [`orca ws clone`](../CLI/orca_ws_clone.md) work across the whole workspace. They accept `-w`, and otherwise use the workspace of the project you're in, or the current workspace.
