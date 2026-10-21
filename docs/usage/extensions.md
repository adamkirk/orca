# Extensions

An extension is a named step that orca can run, defined in a project's [`orca.project.yaml`](../configuration/project_config.md), so the whole team gets the same shortcuts and setup steps. This page covers running them. For every type of extension, and its properties, see the [extensions reference](../configuration/extensions.md).

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
- With [shell completion](./shell_completion.md) installed, `orca ext <tab>` lists the extensions you can run, with their `description`s.

## Other types

Not every extension runs in a container. Without a `service`, a command runs on your machine, in the project's directory. A [`syncFile`](../configuration/extensions.md#syncfile) creates a file from a committed example, and a [`symlink`](../configuration/extensions.md#symlink) links in another project, such as a [library](./workspaces.md#library-projects):

```yaml
extensions:
  - name: install
    command: npm ci
  - name: env
    type: syncFile
    source: .env.example
    target: .env
  - name: link-sdk
    type: symlink
    source: "{{ .Projects.sdk.Path }}"
    target: node_modules/@acme/sdk
```

```sh
$ orca ext install         # npm ci, in the project's directory
$ orca ext env             # creates .env, unless it already exists
$ orca ext link-sdk        # links node_modules/@acme/sdk to wherever sdk is cloned
```

`syncFile` and `symlink` take no arguments, and are safe to run again: they leave an existing `.env`, or a link that's already right, alone.

A library has no services, so its extensions always run on your machine, in the library's directory:

```sh
$ orca ext -p sdk build
```

## Extensions belong to projects

Every extension is defined in a project's `orca.project.yaml`, and runs in that project's directory. The [workspace config](../configuration/workspace_config.md) has none, so `orca ext` needs a project: run it from inside one, or pass `-p`.

A step that several projects need, like installing their dependencies, is defined in each of them, so each project says how it's done:

```yaml
# api/orca.project.yaml
extensions:
  - name: install-deps
    command: npm ci
```

```yaml
# billing/orca.project.yaml
extensions:
  - name: install-deps
    command: composer install
```

Steps for the whole workspace, like seeding the databases, belong to whichever project owns them, often the workspace repository's own. The workspace's [`init.post`](./hooks.md#the-workspaces-initpost) hook can run one once everything is set up.

## Trusting extensions

An extension runs on your machine, from config that someone else wrote, so orca asks before a [hook](./hooks.md) runs one that you haven't asked for yourself:

```
This workspace wants to run the following on your machine:

┌───────────┬────────────┬───────────┬──────────┬─────────────────────┐
│  Project  │    Hook    │ Extension │   Type   │        Runs         │
├───────────┼────────────┼───────────┼──────────┼─────────────────────┤
│ api       │ clone.post │ env       │ syncFile │ .env.example → .env │
│ local-env │ init.post  │ seed      │ command  │ ./scripts/seed.sh   │
└───────────┴────────────┴───────────┴──────────┴─────────────────────┘

Trust these and continue? [y/N]
```

- **Each extension is asked about once.** Your answer is kept in `~/.orca/trust.yaml`, so orca only asks again when the extension's config changes, and then only about that extension, showing what it was before. Changing only its `description` doesn't count.
- **An extension is trusted once,** however many of its project's hooks, or the workspace's `init.post`, run it.
- **Running one yourself, with `orca ext`, counts as trusting it,** so you're never asked about it afterwards.
- **Answering no runs nothing,** and the command fails.
- **`--yes` runs everything without asking,** for that run only: it doesn't record anything. Without a terminal to ask in, e.g. in a script, orca fails rather than waiting, and suggests `--yes`.

