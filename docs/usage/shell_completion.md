# Shell completion

Orca can complete its commands and flags when you press <kbd>Tab</kbd>. [`orca completion`](../CLI/orca_completion.md) generates the script your shell needs. Install it once, using the steps for your shell below.

You only install the script once. It asks orca what to offer each time you press <kbd>Tab</kbd>, so completion stays up to date after [`orca sys self-update`](./system.md#updating) without reinstalling it.

## bash

Completion needs version 2 of the `bash-completion` package. Many Linux distributions install it by default. On macOS, install it with Homebrew, and follow the instructions it prints for loading it from `~/.bash_profile`:

```sh
$ brew install bash-completion@2
```

Then install orca's script:

=== "macOS"

    ```sh
    $ orca completion bash > "$(brew --prefix)/etc/bash_completion.d/orca"
    ```

=== "Linux"

    ```sh
    $ mkdir -p ~/.local/share/bash-completion/completions && orca completion bash > ~/.local/share/bash-completion/completions/orca
    ```

    This installs it for your user only. To install it for everyone, write it to `/etc/bash_completion.d/orca` instead, with `sudo`.

Start a new shell for it to take effect. To try it in the current shell without installing it, run `source <(orca completion bash)`.

## zsh

Write the script to a directory on your `fpath`, as a file named `_orca`. Which directory depends on how your zsh is set up:

=== "oh-my-zsh"

    oh-my-zsh already adds `$ZSH_CUSTOM/completions` to your `fpath`, and enables completion, but doesn't create the directory:

    ```sh
    $ mkdir -p "$ZSH_CUSTOM/completions" && orca completion zsh > "$ZSH_CUSTOM/completions/_orca"
    ```

=== "Homebrew's zsh"

    Homebrew's `site-functions` directory is already on its zsh's `fpath`:

    ```sh
    $ orca completion zsh > "$(brew --prefix)/share/zsh/site-functions/_orca"
    ```

=== "Anything else"

    Create a directory for your own completions:

    ```sh
    $ mkdir -p ~/.zsh/completions && orca completion zsh > ~/.zsh/completions/_orca
    ```

    Then add it to your `fpath` in `~/.zshrc`, before completion is enabled. If `~/.zshrc` doesn't already run `compinit`, add that too:

    ```sh
    fpath=(~/.zsh/completions $fpath)
    autoload -U compinit && compinit
    ```

Start a new shell for it to take effect. To try it in the current shell without installing it, run `source <(orca completion zsh)`.

If orca still doesn't complete, zsh may be using a cached list of completions. Delete the cache and start a new shell:

```sh
$ rm -f ~/.zcompdump*
```

## fish

fish loads completions from `~/.config/fish/completions` automatically:

```sh
$ mkdir -p ~/.config/fish/completions && orca completion fish > ~/.config/fish/completions/orca.fish
```

New shells pick it up straight away. To try it in the current shell, run `orca completion fish | source`.

## PowerShell

Add this line to your PowerShell profile, which you can open with `notepad $PROFILE`:

```powershell
orca completion powershell | Out-String | Invoke-Expression
```

## What completes

| Where | Offers |
| ----- | ------ |
| `orca <tab>` | Commands, with a short description in zsh, fish and PowerShell. |
| `orca g <tab>` | Subcommands of a command group. |
| `orca up --<tab>` | The command's flags. |
| `orca <tab>`, for [plugin](../plugins/index.md) commands | Plugin commands, with `(plugin: <name>)` after their description, and their subcommands and flags. See below. |
| `-w <tab>`, on any command with it | Your registered workspaces. |
| `-p <tab>`, on any command with it | The projects in the workspace, worked out as the command would: from `-w`, if you've typed it, otherwise from your current directory, or your current workspace. See [Choosing a workspace and project](../getting_started/context.md). For `up`, `down`, `restart`, `logs`, `exec` and `debug`, [library projects](./workspaces.md#library-projects) are left out, as those commands don't apply to them. |
| `orca ws switch <tab>` | Your registered workspaces. |
| `orca ws enable <tab>` | The workspace's [disabled](./workspaces.md#enabling-and-disabling-projects) projects, leaving out any you've already typed. |
| `orca ws disable <tab>` | The workspace's enabled projects, leaving out any you've already typed. |
| `orca ext <tab>` | The [extensions](./extensions.md) you can run: the project's, then the workspace's that the project doesn't override. Each has its `description`, in shells that show descriptions. |
| `orca g co <tab>`, `orca g branches <tab>` | Branches in the repository, apart from the current one: local branches first, then those only on a remote, without the remote's name (`feature/x`, not `origin/feature/x`). Remote branches are those git already knows about, so fetch to see new ones. With `--all`, nothing is offered, as the branches of one repository would be misleading for every project. |
| `orca exec <tab>`, `orca logs <tab>` | The services in the project's compose file (`composeFiles.primary`). Docker isn't needed, so this works when it isn't running. |

Anything that can't be worked out, such as projects outside any workspace, or with an invalid config, offers nothing. Orca's own commands never offer file names for their arguments, as they're always names.

To show only names, without descriptions, generate the script with `--no-descriptions`, e.g. `orca completion zsh --no-descriptions`.

### Plugin commands

Starting every plugin on each <kbd>Tab</kbd> would be far too slow, so completion uses the commands plugins provided the last time orca ran a command, which it keeps in `~/.orca/cache/plugin-commands.json`. So:

- After installing, updating or removing a plugin, run any orca command, such as `orca plugins ls`, to update what completes.
- On a fresh install, plugin commands aren't offered until orca has run a command once.
- Only command and flag names complete, not the values plugins expect for their arguments.

The cache is safe to delete. See [The command cache](../plugins/how_it_works.md#the-command-cache).

## Troubleshooting

To see what orca offers for a command line, without your shell involved, run the hidden `__complete` command with the words you've typed, followed by an empty argument for the word being completed:

```sh
$ orca __complete g ""
branches	Searches branches for the git repository.
co	Checkout a branch for a git repository.
...
:4
```

The last line is a code for the shell, and not offered. If this lists what you expect, but your shell doesn't, the script isn't installed or loaded. Check the steps for your shell above, and start a new shell.
