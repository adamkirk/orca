# Extensions

An extension is a named step that orca can run, defined in a project's [`orca.project.yaml`](../configuration/project_config.md) or the [`orca.workspace.yaml`](../configuration/workspace_config.md), so the whole team gets the same shortcuts and setup steps. This page covers running them. For every type of extension, and its properties, see the [extensions reference](../configuration/extensions.md).

The most common is a command that runs in one of the project's services. It saves everyone remembering the service name and the exact command.

```yaml
extensions:
  - name: artisan
    service: app
    command: php artisan
    defaultArgs:
      - list
    description: Run an artisan command
  - name: composer
    service: app
    command: composer
    description: Run composer
```

Run them with [`orca ext`](../CLI/orca_ext.md), from inside the project (or with `-p`):

```sh
$ orca ext artisan                  # php artisan list
$ orca ext artisan migrate          # php artisan migrate
$ orca ext composer require foo/bar # composer require foo/bar
$ orca ext artisan migrate --force  # php artisan migrate --force
```

- Arguments after the extension's name are appended to its `command`. If you don't give any, its `defaultArgs` are used instead.
- Everything after the extension's name is passed on as it is, flags included, like `--force` above. So orca's own flags, like `-p`, must come before the name: `orca ext -p api artisan migrate`. This is the same as [`orca exec`](./environments.md#running-commands-in-a-service) and its service.
- Earlier versions needed `--` before flags meant for the extension. Drop it: a `--` straight after the name is an error.
- Like [`orca exec`](./environments.md#running-commands-in-a-service), the command runs in the existing container if the service is running, or in a throwaway one (`docker compose run --rm`) if not.
- `command` is split on spaces, so it can't contain quoted arguments. Use a script in the container for anything more involved.
- With [shell completion](./shell_completion.md) installed, `orca ext <tab>` lists the project's extensions, with their `description`s.

!!! note
    So far, `orca ext` only runs `command` extensions with a `service`. Commands on your machine, [`syncFile`](../configuration/extensions.md#syncfile) and [`symlink`](../configuration/extensions.md#symlink) can already be defined, but running them is coming soon.
