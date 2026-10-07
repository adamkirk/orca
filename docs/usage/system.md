# Orca itself

Commands for managing your orca installation.

## Checking your setup

```sh
$ orca sys check
```

[`orca sys check`](../CLI/orca_sys_check.md) checks that orca's requirements are installed and working: `docker`, `docker compose` and `git`.

## Updating

```sh
$ orca sys self-update             # the latest release
$ orca sys self-update --to 0.9.0  # a specific version
```

[`orca sys self-update`](../CLI/orca_sys_self-update.md) downloads a release from GitHub and replaces the running binary. If orca is installed somewhere only root can write to, such as `/usr/local/bin`, run it with `sudo`.

[`orca version`](../CLI/orca_version.md) shows the version you're running, with `--short` for just the version number.

## Installing tools

```sh
$ orca sys install sops
```

[`orca sys install`](../CLI/orca_sys_install.md) downloads a supporting tool into `~/.orca/bin` (or `ORCA_TOOLS_PATH`), rather than installing it system-wide. The available tools are `hostctl` and `sops`.

## Config

| Command | Does |
| ------- | ---- |
| [`orca config path`](../CLI/orca_config_path.md) | Shows the path of the [user config](../configuration/user_config.md) in use. |
| [`orca config show`](../CLI/orca_config_show.md) | Shows the user config, with any overrides from flags and environment variables applied. |

## Shell completion

[`orca completion`](../CLI/orca_completion.md) generates completion scripts for bash, zsh, fish and PowerShell. For example, for zsh:

```sh
$ orca completion zsh > "${fpath[1]}/_orca"
```

Run `orca completion <shell> --help` for instructions for your shell.
