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
    $ mkdir -p ~/.local/share/bash-completion/completions
    $ orca completion bash > ~/.local/share/bash-completion/completions/orca
    ```

    This installs it for your user only. To install it for everyone, write it to `/etc/bash_completion.d/orca` instead, with `sudo`.

Start a new shell for it to take effect. To try it in the current shell without installing it, run `source <(orca completion bash)`.

## zsh

Write the script to a directory on your `fpath`, as a file named `_orca`. Which directory depends on how your zsh is set up:

=== "oh-my-zsh"

    oh-my-zsh already adds `$ZSH_CUSTOM/completions` to your `fpath`, and enables completion:

    ```sh
    $ mkdir -p "$ZSH_CUSTOM/completions"
    $ orca completion zsh > "$ZSH_CUSTOM/completions/_orca"
    ```

=== "Homebrew's zsh"

    Homebrew's `site-functions` directory is already on its zsh's `fpath`:

    ```sh
    $ orca completion zsh > "$(brew --prefix)/share/zsh/site-functions/_orca"
    ```

=== "Anything else"

    Create a directory for your own completions:

    ```sh
    $ mkdir -p ~/.zsh/completions
    $ orca completion zsh > ~/.zsh/completions/_orca
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
$ orca completion fish > ~/.config/fish/completions/orca.fish
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

To show only names, without descriptions, generate the script with `--no-descriptions`, e.g. `orca completion zsh --no-descriptions`.

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
