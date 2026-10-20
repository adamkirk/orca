# Project config (`orca.project.yaml`)

Every service project needs an `orca.project.yaml` in the root of its directory, next to its compose file. It tells orca how to run the project. All paths in it are relative to the project's directory.

A [library project](../usage/workspaces.md#library-projects) doesn't need one. It can have one, for extensions, but it can't contain `composeFiles`, `envFiles`, `hosts` or `tlsCerts`, or `command` extensions with a `service`, as a library has no compose stack. Orca lists any it finds as invalid.

## Example

```yaml
composeFiles:
  primary: docker-compose.yml

envFiles:
  - path: .env

hosts:
  - api.acme.test

tlsCerts:
  - "*.acme.test"

extensions:
  - name: env
    type: syncFile
    source: .env.example
    target: .env
  - name: artisan
    service: app
    command: php artisan
    defaultArgs:
      - list
  - name: test
    service: app
    command: vendor/bin/phpunit
```

## Reference

| Property | Description |
| -------- | ----------- |
| [`composeFiles`](#compose-files) | The compose file for the project. |
| [`envFiles`](#env-files) | Env files passed to docker compose. |
| [`hosts`](#hosts) | Hostnames this project serves, added to `/etc/hosts` by [`orca hosts`](../CLI/orca_hosts.md). |
| [`tlsCerts`](#tls-certificates) | Domains to generate TLS certificates for with [`orca tls gen`](../CLI/orca_tls_gen.md). |
| [`extensions`](#extensions) | Named steps, such as commands and files to set up, run with [`orca ext`](../CLI/orca_ext.md). |
| [`hooks`](#hooks) | Extensions to run around operations, such as after the project is cloned. |

### Compose files

```yaml
composeFiles:
  primary: docker-compose.yml
```

`primary` is the compose file orca passes to `docker compose`, along with the [overlay](../compose_files/overlays/index.md) it generates. Keep this a normal compose file: orca makes its changes in the overlay, so the file still works with plain `docker compose`.

!!! note "Not implemented yet"
    The config also accepts `composeFiles.extras` (extra compose files, optionally loaded only `when` conditions on OS, architecture or a property match) and a top-level `properties` list. Both are parsed but not used yet, so they currently have no effect.

### Env files

```yaml
envFiles:
  - path: .env
  - path: .env.local
```

Each file is passed to docker compose with `--env-file`, in order. Compose uses them for [variable interpolation](https://docs.docker.com/compose/how-tos/environment-variables/variable-interpolation/) in the compose file, e.g. `${API_PORT}`. They aren't automatically passed into containers; use `env_file` in your compose file for that.

!!! warning
    When any `--env-file` is given, docker compose no longer loads the project's `.env` file automatically. If you rely on `.env`, list it here.

### Hosts

```yaml
hosts:
  - api.acme.test
  - admin.acme.test
```

Hostnames that should resolve to your machine. [`orca hosts`](../CLI/orca_hosts.md) adds every host from every project in the workspace to `/etc/hosts`, pointing at `127.0.0.1`. See [Hosts & TLS](../usage/hosts_and_tls.md).

### TLS certificates

```yaml
tlsCerts:
  - "*.acme.test"
  - acme.test
```

Domains (wildcards allowed) to generate certificates for, signed by orca's local certificate authority. Quote wildcards, as YAML treats a leading `*` specially. Services can have the certificates mounted with the [TLS injection overlay](../compose_files/overlays/TLS-injection.md). See [Hosts & TLS](../usage/hosts_and_tls.md).

### Extensions

```yaml
extensions:
  - name: artisan
    service: app
    command: php artisan
    defaultArgs:
      - list
    description: Run an artisan command
  - name: env
    type: syncFile
    source: .env.example
    target: .env
```

Named steps, each of a `type`: a `command` (in a service, or on your machine), a `syncFile` or a `symlink`. Paths are relative to the project's directory. Each type takes its own properties, and the config is invalid if an extension has one its type doesn't take.

See the [extensions reference](./extensions.md) for every type and property, and [Extensions](../usage/extensions.md) for running them.

### Hooks

```yaml
hooks:
  clone:
    post:
      - extension: env
      - extension: install-deps
        scope: workspace
  up:
    pre:
      - extension: install-deps
        scope: workspace
```

The extensions to run around an operation, by name, keyed by the operation, then `pre` or `post`. A project can have `clone.post`, `up.pre`, `up.post`, `down.pre` and `down.post`, and a library only `clone.post`. Each entry's `extension` must be defined in this config, or in the workspace config with `scope: workspace`.

See [Hooks](../usage/hooks.md).
