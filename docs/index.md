![coverage](https://raw.githubusercontent.com/adamkirk/orca/badges/.badges/main/coverage.svg)

# Home

Whalecome!

Orca orchestrates development environments that are spread across several repositories. Each repository keeps its own standard `docker-compose.yml`, and orca starts, stops and connects them as a single unit. You also get helpers for working with services, and with git across all of the repositories at once.

## What orca does

| Feature | Summary |
| ------- | ------- |
| [Workspaces & projects](./getting_started/index.md) | Group repositories (projects) into a workspace, then clone and register them all with one command. |
| [Environments](./usage/environments.md) | `up`, `down`, `restart`, `logs` and `exec` against one project or the whole workspace, starting projects in dependency order. |
| [Overlays](./compose_files/overlays/index.md) | Generated compose files that join projects onto a shared network, add DNS aliases and inject TLS certificates, without editing your compose files. |
| [Extensions](./usage/extensions_and_provisioners.md) | Project-defined shortcuts for commands you run in a service, e.g. `orca ext artisan migrate`. |
| [Provisioners](./usage/extensions_and_provisioners.md#provisioners) | One-off setup for a project, such as creating `.env` files from their examples. |
| [Hosts & TLS](./usage/hosts_and_tls.md) | Add your projects' hostnames to `/etc/hosts`, and generate locally trusted TLS certificates for them. |
| [Git helpers](./usage/git.md) | Check out, pull, push and see the status of branches across every project in a workspace. |
| [Plugins](./plugins/index.md) | Add your own commands to orca, as separate programs that orca talks to over gRPC. |

If you're new to orca, start with [Getting started](./getting_started/index.md), then follow [Setting up a workspace](./guides/setting_up_a_workspace.md).

## Initial installation

Head over to the [latest github release](https://github.com/adamkirk/orca/releases/latest), and find the URL for the archive that best suits your system. Put into the script below, and you're good to go!

```sh
$ curl -o /tmp/orca.tar.gz -L "{url}"
$ (cd /tmp && tar -xzvf /tmp/orca.tar.gz)
$ chmod +x /tmp/orca
$ sudo mv /tmp/orca /usr/local/bin/orca
```

Optionally, clean up the files from `/tmp`...

```
$ rm /tmp/orca.tar.gz /tmp/LICENSE /tmp/README.md
```

Then check that orca can find everything it needs (docker, docker compose and git):

```sh
$ orca sys check
```

### Updating

!!! tip "> 0.5.0"
    If you're using a version of at least 0.5.0, you can simply use the [self-update command](./CLI/orca_sys_self-update.md). Run this to replace the currently installed binary with the latest version from github.

If you're on a version below this, then follow the same instructions as above, but `rm /usr/local/bin/orca`, first.

## CLI reference

Every command, with all of its flags, is documented in the [CLI reference](./CLI/orca.md). Those pages are generated from the CLI itself, so they always match the released version.
