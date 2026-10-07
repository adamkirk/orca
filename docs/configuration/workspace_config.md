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

  - name: web
    repository:
      ssh: git@github.com:acme/web.git
    requires:
      - api
```

## Reference

### Top level

| Property | Description |
| -------- | ----------- |
| `name` | The workspace's name. It's used in commands (`-w acme`), compose project names (`orca-acme-api`) and DNS aliases, so keep it short and lowercase. |
| `projects` | The projects in this workspace, see [below](#projects). |
| `overlays` | Workspace-wide [overlays](../compose_files/overlays/index.md). Currently only [`network`](../compose_files/overlays/network.md) is available. |

### Projects

| Property | Description |
| -------- | ----------- |
| `name` | The project's name, unique within the workspace. When cloning, it's also the name of the directory the project is cloned into. |
| `repository` | Where the project's code comes from, see [below](#repository). |
| `requires` | Names of other projects in this workspace that must be started before this one. |

### Repository

Each project needs exactly one of these:

| Property | Description |
| -------- | ----------- |
| `ssh` | The git URL to clone the project from, e.g. `git@github.com:acme/api.git`. |
| `self` | Set to `true` if the project is the repository containing this workspace config. Nothing is cloned; the repository's root is registered as the project's directory. |

## Dependencies between projects

`requires` controls the order projects are started and stopped in, when you run a command against a whole workspace:

- [`orca up`](../CLI/orca_up.md) starts projects with no requirements first, then the projects that require them, and so on.
- [`orca down`](../CLI/orca_down.md) does the reverse, stopping dependent projects before the ones they require.

In the example above, `up` starts `local-env`, then `api`, then `web`, and `down` stops them in the opposite order.

`requires` must not contain cycles. It isn't used when you run a command against a single project: `orca up -p web` starts only `web`, not `api` or `local-env`.

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
