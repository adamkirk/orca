# Configuration

Orca uses three config files, each with a different owner:

| File | Lives in | Written by | Purpose |
| ---- | -------- | ---------- | ------- |
| [`orca.yaml`](./user_config.md) | `~/.orca/` | Orca (you can edit it too) | Your machine: which workspaces you have, where each project is cloned, the current workspace, logging and plugin directories. |
| [`orca.workspace.yaml`](./workspace_config.md) | The workspace repository | You, shared with your team | The projects in a workspace, where to clone them from, how they depend on each other, and workspace-wide overlays. |
| [`orca.project.yaml`](./project_config.md) | Each project's repository | You, shared with your team | How to run one project: its compose file, env files, hostnames, TLS certificates, extensions and provisioners. |

The workspace and project files are meant to be committed alongside your code, so everyone gets the same environment. The user config is specific to your machine.

## The `~/.orca` directory

| Path | Contents |
| ---- | -------- |
| `~/.orca/orca.yaml` | Your [user config](./user_config.md). |
| `~/.orca/overlays/{workspace}/{project}.yaml` | Generated [overlay](../compose_files/overlays/index.md) compose files. Safe to delete, they're regenerated as needed. |
| `~/.orca/tls/` | The root certificate authority (`cert.pem`, `key.pem`) and generated certificates (`certs/`). See [Hosts & TLS](../usage/hosts_and_tls.md). |
| `~/.orca/bin/` | Tools installed with [`orca sys install`](../CLI/orca_sys_install.md). |
| `~/.orca/plugins/` | The default directory orca loads [plugins](../plugins/index.md) from. |
