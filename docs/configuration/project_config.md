# Project config (`orca.project.yaml`)

Every project needs an `orca.project.yaml` in the root of its directory, next to its compose file. It tells orca how to run the project. All paths in it are relative to the project's directory.

## Example

```yaml
composeFiles:
  primary: docker-compose.yml

envFiles:
  - path: .env

provisioners:
  - exampleFile:
      src: .env.example
      target: .env
      type: dotenv

hosts:
  - api.acme.test

tlsCerts:
  - "*.acme.test"

extensions:
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
| [`provisioners`](#provisioners) | One-off setup steps, run with [`orca provision`](../CLI/orca_provision.md). |
| [`hosts`](#hosts) | Hostnames this project serves, added to `/etc/hosts` by [`orca hosts`](../CLI/orca_hosts.md). |
| [`tlsCerts`](#tls-certificates) | Domains to generate TLS certificates for with [`orca tls gen`](../CLI/orca_tls_gen.md). |
| [`extensions`](#extensions) | Named commands that run in one of the project's services, with [`orca ext`](../CLI/orca_ext.md). |

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

### Provisioners

```yaml
provisioners:
  - exampleFile:
      src: .env.example
      target: .env
      type: dotenv
```

Provisioners do one-off setup that a fresh clone needs. They run with [`orca provision`](../CLI/orca_provision.md). The only provisioner so far is `exampleFile`, which copies `src` to `target` if `target` doesn't exist yet, keeping the file's permissions. An existing `target` is never overwritten, so it's safe to run repeatedly. `type` is currently always `dotenv`.

See [Extensions & provisioners](../usage/extensions_and_provisioners.md#provisioners).

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
```

| Property | Description |
| -------- | ----------- |
| `name` | What you type after `orca ext`, e.g. `orca ext artisan`. |
| `service` | The compose service to run the command in. Required. |
| `command` | The command to run. It's split on spaces, so quoting isn't supported. |
| `defaultArgs` | Arguments added when you don't give any. Any arguments you do give replace these entirely. |

See [Extensions & provisioners](../usage/extensions_and_provisioners.md#extensions).
