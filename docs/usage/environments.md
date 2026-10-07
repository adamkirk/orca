# Environments

These commands run docker compose for you, against one project or a whole workspace. Each one works out what to act on from your current directory, or from `-w` and `-p`. See [Choosing a workspace and project](../getting_started/context.md).

## How orca runs compose

For each project, orca runs `docker compose` from the project's directory with:

- the project's `composeFiles.primary`,
- the [overlay](../compose_files/overlays/index.md) orca generates for it,
- each of the project's `envFiles`, as `--env-file`,
- a compose project name of `orca-{workspace}-{project}`.

To see the exact command, run [`orca debug show-compose-command`](../CLI/orca_debug_show-compose-command.md) inside a project. It also works as a prefix for any compose command orca doesn't wrap:

```sh
$ $(orca debug show-compose-command) ps
$ $(orca debug show-compose-command) pull
```

[`orca debug show-compose-config`](../CLI/orca_debug_show-compose-config.md) shows the final config, after your compose file, the overlay and env files are merged together.

## Starting and stopping

```sh
$ orca up                 # this project, or the whole current workspace
$ orca down
$ orca restart            # down, then up
```

| Command | Runs |
| ------- | ---- |
| [`orca up`](../CLI/orca_up.md) | `docker compose up -d` |
| [`orca down`](../CLI/orca_down.md) | `docker compose down --remove-orphans` |
| [`orca restart`](../CLI/orca_restart.md) | `down`, then `up` |

For a whole workspace, `up` starts projects in dependency order (projects before the ones that `require` them), and `down` stops them in reverse. See [dependencies between projects](../configuration/workspace_config.md#dependencies-between-projects).

## Logs

```sh
$ orca logs               # every service in this project
$ orca logs -s app        # just the "app" service
```

[`orca logs`](../CLI/orca_logs.md) follows a single project's logs (`docker compose logs -f`). Press Ctrl-C to stop.

## Running commands in a service

```sh
$ orca exec -s app sh
$ orca exec -s app -- ls -la /app
```

[`orca exec`](../CLI/orca_exec.md) runs a command in one of a project's services:

- If the service is running, it uses `docker compose exec`.
- If it isn't, it uses `docker compose run --rm`, starting a throwaway container.

Either way you get an interactive terminal. Put `--` before the command if it has flags of its own, so orca doesn't try to parse them.

For commands you run often, define an [extension](./extensions_and_provisioners.md#extensions) instead.
