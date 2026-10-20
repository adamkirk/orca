# Extensions

An extension is a named step that orca can run: a command, or a change to a file. It's defined in the `extensions` list of a [project config](./project_config.md) or the [workspace config](./workspace_config.md), and run by name with [`orca ext`](../CLI/orca_ext.md). See [Extensions](../usage/extensions.md) for how to use them.

```yaml
extensions:
  - name: artisan             # a command, in one of the project's services
    service: app
    command: php artisan
    defaultArgs: [list]
    description: Run an artisan command

  - name: install             # a command, on your machine
    command: npm ci

  - name: env                 # a file created from a committed example
    type: syncFile
    source: .env.example
    target: .env

  - name: link-sdk            # a symlink to another project
    type: symlink
    source: "{{ .Projects.sdk.Path }}"
    target: node_modules/@acme/sdk
```

## Shared properties

Every type of extension takes these:

| Property | Required | Description |
| -------- | -------- | ----------- |
| `name` | Yes | What you type after `orca ext`, e.g. `orca ext artisan`. Unique within the config it's defined in. |
| `type` | No | One of the [types below](#types). Defaults to `command`, so a command doesn't need it. |
| `description` | No | Shown next to the name when you [complete](../usage/shell_completion.md) `orca ext <tab>`, in shells that show descriptions. |

Every other property depends on the type. A property that the type doesn't take is an error, rather than being ignored. That way, a typo, or a property meant for a different type, is caught when the config loads.

## Paths

Paths are relative to the directory the extension runs in. That's the project's directory, for a project's extension, and the directory of `orca.workspace.yaml`, for a workspace's.

A `source` can refer to where another project is cloned with `{{ .Projects.<name>.Path }}`, e.g. `{{ .Projects.sdk.Path }}`. Any project name works, including ones with a `-`, like `{{ .Projects.acme-sdk.Path }}/dist`. It fails if the project isn't in the workspace, or isn't cloned yet. This is how a [library](../usage/workspaces.md#library-projects) is linked into another project.

## Types

### `command`

**What it does:** runs a command, either in one of the project's compose services, or on your machine.

**Properties:**

| Property | Required | Default | Description |
| -------- | -------- | ------- | ----------- |
| `command` | Yes | | The command to run. It's split on spaces, so quoting isn't supported. Use a script for anything more involved. |
| `service` | No | | The compose service to run the command in. Without it, the command runs on your machine. A [library](../usage/workspaces.md#library-projects) has no services, so can't use it. |
| `defaultArgs` | No | `[]` | Arguments added when you don't give any. Any arguments you do give replace them entirely. |
| `chdir` | No | | A directory to run the command in, relative to the one the extension runs in. Only used on your machine. |

**Behaviour:**

- In a service, the command runs in the existing container if the service is running, or in a throwaway one (`docker compose run --rm`) if not.
- On your machine, it runs in the project's directory (or the workspace's), plus `chdir`.
- Orca can't tell whether a command is safe to run again, so write commands that are. For example, `npm ci` rather than a script that fails when its work is already done.

**Example:**

```yaml
- name: migrate
  service: app
  command: php artisan migrate
- name: install
  command: npm ci
  chdir: frontend
```

### `syncFile`

**What it does:** creates a file from a committed example, such as each developer's own, uncommitted `.env`.

**Properties:**

| Property | Required | Default | Description |
| -------- | -------- | ------- | ----------- |
| `source` | Yes | | The file to copy, e.g. `.env.example`. |
| `target` | Yes | | The file to create, e.g. `.env`. |

**Behaviour:**

- If `target` doesn't exist, `source` is copied to it, keeping its permissions.
- If `target` already exists, it's left alone, so your own changes to it are never overwritten. That also makes it safe to run again.

**Example:**

```yaml
- name: env
  type: syncFile
  source: .env.example
  target: .env
```

### `symlink`

**What it does:** creates a symlink, typically to link a library project into another project.

**Properties:**

| Property | Required | Default | Description |
| -------- | -------- | ------- | ----------- |
| `source` | Yes | | What the link points to. Usually another project, e.g. `{{ .Projects.sdk.Path }}`. |
| `target` | Yes | | Where to create the link, e.g. `node_modules/@acme/sdk`. |

**Behaviour:**

- `target` is created as an absolute symlink to `source`, along with any directories it needs.
- If `target` is already a symlink to `source`, nothing changes, so it's safe to run again.
- If `target` is anything else, the extension fails, rather than replacing it.

**Example:**

```yaml
- name: link-sdk
  type: symlink
  source: "{{ .Projects.sdk.Path }}"
  target: node_modules/@acme/sdk
```

## Validation

Orca checks extensions whenever it loads the config they're in, and lists every problem it finds, e.g.:

```
Project config '/Users/me/code/acme/api/orca.project.yaml' is invalid:
  - extension 'env': 'service' can't be used with type 'syncFile' (it takes: source, target)
  - extension 'link-sdk': 'target' is required for type 'symlink'
  - extension 'tidy': unknown type 'copy' (expected command, symlink, syncFile)
```
