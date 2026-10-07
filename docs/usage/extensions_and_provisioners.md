# Extensions & provisioners

Both are defined in a project's [`orca.project.yaml`](../configuration/project_config.md), so the whole team gets the same shortcuts and setup steps.

## Extensions

An extension is a named command that runs in one of the project's services. It saves everyone remembering the service name and the exact command.

```yaml
extensions:
  - name: artisan
    service: app
    command: php artisan
    defaultArgs:
      - list
  - name: composer
    service: app
    command: composer
```

Run them with [`orca ext`](../CLI/orca_ext.md), from inside the project (or with `-p`):

```sh
$ orca ext artisan                  # php artisan list
$ orca ext artisan migrate          # php artisan migrate
$ orca ext composer require foo/bar # composer require foo/bar
```

- Arguments after the extension's name are appended to its `command`. If you don't give any, its `defaultArgs` are used instead.
- Like [`orca exec`](./environments.md#running-commands-in-a-service), the command runs in the existing container if the service is running, or in a throwaway one (`docker compose run --rm`) if not.
- `command` is split on spaces, so it can't contain quoted arguments. Use a script in the container for anything more involved.

!!! note
    `service` is required: extensions that run on your machine rather than in a container aren't supported yet. Extensions also accept a `chdir` property, but it isn't used yet.

## Provisioners

Provisioners handle the one-off setup a fresh clone of a project needs. Run them with [`orca provision`](../CLI/orca_provision.md), from inside the project's directory:

```sh
$ cd ~/code/acme/api
$ orca provision
```

They're safe to run more than once: each provisioner skips work that's already been done.

### `exampleFile`

```yaml
provisioners:
  - exampleFile:
      src: .env.example
      target: .env
      type: dotenv
```

Copies `src` to `target`, unless `target` already exists. This is the usual way to give each developer their own, uncommitted `.env` based on a committed example. Paths are relative to the project directory, and `type` is currently always `dotenv`.

!!! note
    `orca provision` doesn't accept `-w` or `-p` yet, so run it from inside the project's directory.
