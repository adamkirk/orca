# Preferences (`orca.preferences.yaml`)

The preferences file holds your own choices for a workspace, such as which projects you want to run. Everyone in the team gets every project cloned, but each developer chooses which of them are enabled.

## Example

```yaml
# orca.preferences.yaml
projects:
  api:
    enabled: true
  web:
    enabled: false
```

## Reference

| Property | Description |
| -------- | ----------- |
| `projects.<name>.enabled` | Whether the project runs in workspace-wide commands, `true` or `false`. Without it, the project follows the [workspace config's `enabled`](./workspace_config.md#projects), which enables it unless it says otherwise. |

## Defaults from the workspace config

The workspace config can set each project's `enabled`, as the team's default, e.g. to leave a rarely used project off for everyone:

```yaml
# orca.workspace.yaml
projects:
  - name: admin
    repository:
      ssh: git@github.com:acme/admin.git
    enabled: false
```

Your preferences file overrides it, where you've set `enabled` for that project. So whether a project is enabled comes from, in order:

1. `enabled` in your preferences file, if it's set.
2. `enabled` in the workspace config, if it's set.
3. Otherwise, it's enabled.

What being disabled means for each command is covered in [Enabling and disabling projects](../usage/workspaces.md#enabling-and-disabling-projects).

## Where it lives

By default the file is `orca.preferences.yaml`, in the same directory as `orca.workspace.yaml`. The workspace config can choose another location, relative to itself:

```yaml
# orca.workspace.yaml
preferences:
  path: config/orca.preferences.yaml
```

## Creating and editing it

- [`orca ws init`](../CLI/orca_ws_init.md) creates the file, listing every project without setting `enabled`, so each follows the workspace config until you choose otherwise. It never overwrites a file that already exists.
- [`orca ws enable`](../CLI/orca_ws_enable.md) and [`orca ws disable`](../CLI/orca_ws_disable.md) edit it, creating it if it's missing. Anything else in the file, including comments, is kept.
- You can also edit it by hand.

Without a file, the workspace config decides, so workspaces initialised before preferences existed keep working as they did.

A project that's added to the workspace config later follows its default until you enable or disable it. A project in the file that isn't in the workspace config, e.g. one that's been removed, is ignored, and [`orca ws projects`](../CLI/orca_ws_projects.md) warns about it.

## Committing it

Orca doesn't ignore the file, or change your repository's git config, so whether it's committed is up to your team:

- **Usually it's personal**, so list it in the workspace repository's `.gitignore`. `orca ws init` reminds you to if it isn't ignored.
- **Some teams commit it**, as shared defaults that each developer then changes locally.

## Validation

Orca checks the file whenever it loads the workspace, so every command against the workspace fails if it's invalid. Every problem found is listed, e.g.:

```
Preferences file '/Users/me/code/acme/local-env/orca.preferences.yaml' is invalid:
  - unknown property 'colour'
  - project 'api': 'enabled' must be true or false
```
