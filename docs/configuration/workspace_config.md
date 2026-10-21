# Workspace config (`orca.workspace.yaml`)

The workspace config defines which projects make up a workspace, and where to clone them from. Commit it to a repository your team shares: either a repository just for the environment, or one of the projects themselves (see [`self`](#repository)).

For a full walk-through, see [Setting up a workspace](../guides/setting_up_a_workspace.md).

## Example

```yaml
name: acme

overlays:
  network:
    enabled: true
    createIn: local-env

projects:
  - name: local-env
    repository:
      self: true

  - name: api
    repository:
      ssh: git@github.com:acme/api.git
    requires:
      - local-env

  - name: sdk
    type: library
    repository:
      ssh: git@github.com:acme/sdk.git
    path: api/packages/sdk

  - name: web
    repository:
      ssh: git@github.com:acme/web.git
    requires:
      - api
      - sdk
```

## Reference

### Top level

| Property | Description |
| -------- | ----------- |
| `name` | The workspace's name. It's used in commands (`-w acme`), compose project names (`orca-acme-api`) and DNS aliases, so keep it short and lowercase. |
| `projects` | The projects in this workspace, see [below](#projects). |
| `overlays` | Workspace-wide [overlays](../compose_files/overlays/index.md). Currently only [`network`](../compose_files/overlays/network.md) is available. |
| `preferences.path` | Where each developer's [preferences file](./preferences.md) lives, relative to this file. Defaults to `orca.preferences.yaml`, next to it. |
| `hooks` | Optional. Only [`init.post`](../usage/hooks.md#the-workspaces-initpost), which runs once every project is cloned and set up, e.g. to seed the databases. The workspace config has no extensions, so each entry names a `project` and one of its `extension`s, which runs in that project's directory. Every other hook belongs in a [project's config](./project_config.md#hooks). |

### Projects

| Property | Description |
| -------- | ----------- |
| `name` | The project's name, unique within the workspace. When cloning, it's also the name of the directory the project is cloned into, unless `path` is set. |
| `type` | Optional. `service`, a project with a compose stack to run, or `library`, a repository that's only cloned and set up, never run. Defaults to `service`. See [Library projects](../usage/workspaces.md#library-projects). |
| `repository` | Where the project's code comes from, see [below](#repository). |
| `path` | Optional. Where to clone the project, relative to the directory projects are cloned into, e.g. `api/packages/sdk`. It can be inside another project, but must stay inside that directory, so a workspace's layout is the same on every machine. Defaults to the project's `name`. Can't be combined with `repository.self`. |
| `requires` | Names of other projects in this workspace that must be started before this one. |
| `enabled` | Optional. Whether the project runs in workspace-wide commands by default, `true` or `false`. Defaults to `true`. Each developer can override it in their [preferences file](./preferences.md). |

### Repository

Each project needs exactly one of `ssh` or `self`:

| Property | Description |
| -------- | ----------- |
| `ssh` | The git URL to clone the project from, e.g. `git@github.com:acme/api.git`. |
| `self` | Set to `true` if the project is the repository containing this workspace config. Nothing is cloned; the repository's root is registered as the project's directory. |
| `branch` | Optional. The branch to check out when the project is first cloned. Without it, the remote's default branch is used. Can't be combined with `self`. |

`branch` is only used for the first clone. Orca never switches an existing clone to another branch, or warns that it's on a different branch from its config: which branch you're on is up to you. It's meant for trying orca out on a branch before merging, see [Trying orca on a branch](../guides/setting_up_a_workspace.md#trying-orca-on-a-branch).

## Validation

Orca checks the workspace config whenever it loads it, so every command fails if it's invalid. It checks that:

- `name` is set.
- Every project has a `name`.
- Every project has exactly one of `repository.ssh` or `repository.self`.
- No project has both `repository.self` and `repository.branch`.
- Every `type` is `service` or `library`.
- No `path` is absolute, or escapes the directory projects are cloned into with `..`, and none is combined with `repository.self`.
- `hooks` only has `init.post`, and each of its entries has an `extension` and a `project` that's in the workspace, and nothing else. Once that project is cloned, its config must define the extension.

Every problem found is listed, e.g.:

```
Workspace config '/Users/me/code/acme/local-env/orca.workspace.yaml' is invalid:
  - 'name' is required
  - project 'api': exactly one of 'repository.ssh' or 'repository.self' is required
```

## Dependencies between projects

`requires` controls the order projects are started and stopped in, when you run a command against a whole workspace:

- [`orca up`](../CLI/orca_up.md) starts projects with no requirements first, then the projects that require them, and so on.
- [`orca down`](../CLI/orca_down.md) does the reverse, stopping dependent projects before the ones they require.

In the example above, `up` starts `local-env`, then `api`, then `web`, and `down` stops them in the opposite order.

`requires` must not contain cycles. It isn't used when you run a command against a single project: `orca up -p web` starts only `web`, not `api` or `local-env`.

A project can require a [library](../usage/workspaces.md#library-projects). Libraries are never started, but [`orca ws clone`](../CLI/orca_ws_clone.md) clones them before the projects that require them. If a required library isn't cloned, `orca up` warns, and starts the project anyway.

`requires` also wins over your [preferences](./preferences.md): if an enabled project requires a disabled one, `orca up` starts the disabled one too, with a warning. See [Enabling and disabling projects](../usage/workspaces.md#enabling-and-disabling-projects).

## Overlays

```yaml
overlays:
  network:
    enabled: true
    createIn: local-env
    disableAliases: false
    aliasPattern: "{{ .Service }}.{{ .Project }}.{{ .Workspace }}.local"
```

The network overlay puts every project in the workspace on one shared docker network, so services in different projects can reach each other. It also gives each service a predictable DNS name. See [Network & Aliases](../compose_files/overlays/network.md) for the details of each property.
