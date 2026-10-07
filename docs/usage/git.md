# Git

The `orca g` commands are shortcuts for everyday git tasks. Several of them can act on every project in a workspace at once, which helps when a change spans repositories.

## Across projects

These accept `-a/--all` to act on every project in the workspace, as well as the usual `-w` and `-p` (see [Choosing a workspace and project](../getting_started/context.md)). Without them, they act on the project you're in.

They also work in any git repository, even one that isn't part of a workspace. If orca can't work out a workspace, they act on the repository you're in. Only `--all` and `-p` need a workspace.

| Command | Does |
| ------- | ---- |
| [`orca g status`](../CLI/orca_g_status.md) | Shows a table of each project's current branch and uncommitted changes. |
| [`orca g co [search]`](../CLI/orca_g_co.md) | Checks out a branch matching `search` (see below). |
| [`orca g pull`](../CLI/orca_g_pull.md) | Pulls the current branch from `origin`. `-r/--rebase` passes `--rebase` to `git pull`. |
| [`orca g push`](../CLI/orca_g_push.md) | Pushes the current branch to `origin`, to a branch of the same name. `-f/--force` force pushes. |

```sh
$ orca g status --all
$ orca g co --all feature/payments
$ orca g pull --all --rebase
$ orca g push --all
```

### Checking out branches

`orca g co` searches branch names for the text you give it (case-insensitive):

- If exactly one branch matches, it's checked out.
- If several match, you choose from a list.

| Flag | Effect |
| ---- | ------ |
| `-a`, `--all` | Check out the branch in every project in the workspace. |
| `-b`, `--create` | Create the branch if it doesn't exist. Can't be used with `--pull`. |
| `--pull` | Pull the branch from `origin` after checking it out. |
| `-r`, `--rebase` | With `--pull`, pull with `--rebase`. |

## In the current repository

These act on the git repository you're in:

| Command | Does |
| ------- | ---- |
| [`orca g branches [search]`](../CLI/orca_g_branches.md) | Lists branches containing `search`, or every branch. |
| [`orca g logl`](../CLI/orca_g_logl.md) | Shows the last commits, one line each. `-n` sets how many (default 10). |
| [`orca g rbi`](../CLI/orca_g_rbi.md) | Starts an interactive rebase of the last `-n` commits (default 2). |
| [`orca g undo`](../CLI/orca_g_undo.md) | Removes the last `-n` commits (default 1) from the branch. |

!!! danger "`orca g undo` is destructive"
    It runs `git reset --hard HEAD~n`, which discards those commits **and any uncommitted changes**. It asks for confirmation first, unless you pass `-y`.
