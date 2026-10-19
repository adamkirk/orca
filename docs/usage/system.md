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

## Installing other tools

Orca doesn't install the other tools your projects use. If your team wants to provide them, at the versions it uses, write a [plugin](../plugins/index.md) for it.

Earlier versions had an `orca sys install` command, which downloaded tools into `~/.orca/bin`. Orca no longer uses that directory or `ORCA_TOOLS_PATH`, so you can delete them.

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
