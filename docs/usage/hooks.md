# Hooks

A hook runs [extensions](./extensions.md) around an operation, such as after a project is cloned, or before the stack starts. They're how a team gets a working environment without a setup guide. For example, each project creates its `.env` and links in a library after it's cloned, and the workspace seeds the databases once everything is set up.

A hook only lists extensions by name. What each one does is defined in `extensions`, so every step can also be run on its own with [`orca ext`](../CLI/orca_ext.md), and a step used by several hooks is defined once.

Hooks run automatically: `clone.post` and `init.post` from [`orca ws init`](./workspaces.md#from-a-local-directory) and [`orca ws clone`](./workspaces.md#cloning-projects), and `up` and `down` from [`orca up` and `orca down`](./environments.md#hooks). Any hook can also be [run by hand](#running-a-hook-by-hand).

## Defining hooks

Hooks are keyed by the operation, then `pre` (before it) or `post` (after it). Each entry names an extension:

=== "orca.project.yaml"
    ```yaml
    extensions:
      - name: env
        type: syncFile
        source: .env.example
        target: .env
      - name: link-sdk
        type: symlink
        source: "{{ .Projects.sdk.Path }}"
        target: node_modules/@acme/sdk

    hooks:
      clone:
        post:
          - extension: env
          - extension: link-sdk
          - extension: install-deps   # defined once, in the workspace config
            scope: workspace
      up:
        pre:
          - extension: install-deps
            scope: workspace
    ```
=== "orca.workspace.yaml"
    ```yaml
    extensions:
      - name: install-deps
        command: npm ci
      - name: seed
        command: ./scripts/seed.sh

    hooks:
      init:
        post:
          - extension: seed
    ```

| Property | Description |
| -------- | ----------- |
| `extension` | The name of the extension to run. |
| `scope` | Where the extension is defined: `project` (the project's config) or `workspace` (the workspace config). It defaults to the config the hook is in, so it's only needed for a project's hook that runs one of the workspace's extensions. |

An entry can't define an extension itself. Define it in `extensions`, and reference it by name.

### Which hooks there are

| Hook | Defined in | Runs |
| ---- | ---------- | ---- |
| `clone.post` | project | After the project is cloned, in its directory, by `ws init` and `ws clone`. Again whenever they're re-run. |
| `init.post` | workspace | At the end of `ws init`, after every project is cloned, and their `clone.post` hooks have run. |
| `up.pre`, `up.post` | workspace, project | Before and after the project is started, by `orca up`. The workspace's run before the first project starts, and after the last. See [the order](./environments.md#hooks). |
| `down.pre`, `down.post` | workspace, project | Before and after it's stopped, by `orca down`, in the same way. |

There's no `init.pre` or `clone.pre`. A project's config can't be read until it's cloned, so nothing it defines can run before then.

A [library](./workspaces.md#library-projects) is never started or stopped, so it can only have `clone.post`.

### Where extensions run

Every extension in a project's hook runs in that project's directory, including one defined in the workspace config. So a shared `install-deps` that runs `npm ci` installs each project that uses it. The workspace's own hooks run in the directory of `orca.workspace.yaml`.

## Running a hook by hand

[`orca hooks run`](../CLI/orca_hooks_run.md) runs every extension in a hook, in order:

```sh
$ orca hooks run clone.post -p api     # set api up again
$ orca hooks run init.post             # run the workspace's init.post
```

```
api clone.post: env
  Created .env from .env.example.
api clone.post: install-deps
  added 312 packages in 4s
```

- Inside a project, or with `-p`, it runs the project's hook. Outside of a project, or for `init.post`, which only the workspace has, it runs the workspace's.
- Each step's output is shown as it runs, indented under it.
- Before anything runs, any extension that's new, or has changed since you trusted it, is listed, and you're asked to trust it. See [Trusting extensions](./extensions.md#trusting-extensions). `--yes` runs everything without asking, for that run only.
- With [shell completion](./shell_completion.md) installed, `orca hooks run <tab>` lists the hooks that run something, with the extensions each one runs.

To run one step of a hook on its own, run its extension with `orca ext`, e.g. `orca ext -p api env`.

## When a step fails

A hook stops at the first extension that fails, and the command exits non-zero. Nothing after it runs, not even the other projects' hooks during `ws init`, but nothing that's already been done is undone: every project stays cloned. The error says which step it was, and how to run it again:

```
api clone.post: install-deps
  npm ERR! network request failed
Extension 'install-deps' failed!
  exit status 1
This was 'install-deps', in api's clone.post hook. Nothing after it ran.
Once it's fixed, re-run it with 'orca ext -w acme -p api install-deps', or the whole hook with 'orca hooks run -w acme -p api clone.post'.
```

## Write extensions that are safe to run again

Orca doesn't record which hooks have run. Running `ws init` or `ws clone` again runs every project's hooks again, and that's the way to finish a setup that failed part-way. So every extension a hook runs must be safe to run more than once:

- `syncFile` and `symlink` already are: they leave an existing file, or a link that's already right, alone.
- For a `command`, orca can't tell, so it's up to you. Prefer commands that are naturally safe to repeat (`npm ci`, `composer install`, `make`), and make scripts check whether their work is already done, e.g. a seed script that skips a database that's already seeded.
