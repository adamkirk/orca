# User config (`~/.orca/orca.yaml`)

Your user config records the workspaces you've set up on this machine. Orca creates it on first run, and keeps it up to date as you use [`orca ws`](../CLI/orca_ws.md) commands. You can edit it by hand, e.g. to point at a project you cloned somewhere unusual.

Use [`orca config path`](../CLI/orca_config_path.md) to see which file is in use, and [`orca config show`](../CLI/orca_config_show.md) to see the config after any overrides from flags or environment variables are applied.

## Example

```yaml
logging:
    level: none
    format: text
workspaces:
    - name: acme
      path: /Users/me/code/acme/local-env/orca.workspace.yaml
      projects:
        - name: local-env
          path: /Users/me/code/acme/local-env
        - name: api
          path: /Users/me/code/acme/api
        - name: web
          path: /Users/me/code/acme/web
currentWorkspace: acme
plugins:
    dirs:
        - /Users/me/code/acme/orca-plugins/bin
```

## Reference

| Property | Description | Default |
| -------- | ----------- | ------- |
| `logging.level` | How much orca logs. One of `debug`, `info`, `warn`, `error` or `none`. Most errors are already shown to you directly, so logging is mainly useful for debugging. | `none` |
| `logging.format` | `text` for human-readable logs, anything else for JSON. | `text` |
| `workspaces[].name` | The workspace's name, which must match the `name` in its `orca.workspace.yaml`. | |
| `workspaces[].path` | Absolute path to the workspace's `orca.workspace.yaml`. | |
| `workspaces[].projects[].name` | A project's name, matching its entry in the workspace config. | |
| `workspaces[].projects[].path` | Absolute path to the directory the project is cloned in. Orca uses these paths to work out [which project you're in](../getting_started/context.md). | |
| `currentWorkspace` | The workspace used when you're not inside a project directory and don't pass `-w`. Set with [`orca ws switch`](../CLI/orca_ws_switch.md), cleared with [`orca ws clear-current`](../CLI/orca_ws_clear-current.md). | `""` |
| `plugins.dirs` | Extra directories to load [plugins](../plugins/index.md) from, in addition to `~/.orca/plugins`. | `[]` |

A project listed in a workspace config, but missing from your user config, is treated as not cloned yet. Run [`orca ws clone`](../CLI/orca_ws_clone.md) to clone and register it.

## Overrides

Some settings can be overridden for a single command, without changing the file. The precedence is: flag, then environment variable, then the config file.

| Setting | Flag | Environment variable |
| ------- | ---- | -------------------- |
| `logging.level` | `--log-level` | `ORCA_LOGGING_LEVEL` |
| `logging.format` | `--log-format` | `ORCA_LOGGING_FORMAT` |

These environment variables change where orca looks for things:

| Environment variable | Effect | Default |
| -------------------- | ------ | ------- |
| `ORCA_CONFIG_PATH` | The user config file to use. | `~/.orca/orca.yaml` |
| `ORCA_PLUGINS_PATH` | The default plugins directory (`plugins.dirs` are still searched as well). | `~/.orca/plugins` |
| `ORCA_TOOLS_PATH` | Where [`orca sys install`](../CLI/orca_sys_install.md) puts tools. | `~/.orca/bin` |
