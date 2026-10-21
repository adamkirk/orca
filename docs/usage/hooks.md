# Hooks

A hook runs [extensions](./extensions.md) around an operation, such as after a project is cloned, or before it starts. They're how a team gets a working environment without a setup guide. For example, each project creates its `.env` and links in a library after it's cloned, and the workspace seeds the databases once everything is set up.

A hook only lists extensions by name. What each one does is defined in the same project's `extensions`, so every step can also be run on its own with [`orca ext`](../CLI/orca_ext.md), and a step used by several of the project's hooks is defined once.

Hooks run automatically: `clone.post` and `init.post` from [`orca ws init`](./workspaces.md#from-a-local-directory) and [`orca ws clone`](./workspaces.md#cloning-projects), and `up` and `down` from [`orca up` and `orca down`](./environments.md#hooks). Any hook can also be [run by hand](#running-a-hook-by-hand).

## Defining hooks

A project's hooks are in its `orca.project.yaml`, keyed by the operation, then `pre` (before it) or `post` (after it). Each entry names one of the project's own extensions:

```yaml
# orca.project.yaml
extensions:
  - name: env
    type: syncFile
    source: .env.example
    target: .env
  - name: link-sdk
    type: symlink
    source: "{{ .Projects.sdk.Path }}"
    target: node_modules/@acme/sdk
  - name: install-deps
    command: npm ci

hooks:
  clone:
    post:
      - extension: env
      - extension: link-sdk
      - extension: install-deps
  up:
    pre:
      - extension: install-deps
```

| Property | Description |
| -------- | ----------- |
| `extension` | The name of one of the project's extensions. |

An entry can't define an extension itself, or use another project's. Define it in the project's `extensions`, and reference it by name. A step several projects need, like `install-deps` above, is defined in each of them.

Every extension runs in its project's directory. The workspace repository's own project (the one with [`self: true`](../configuration/workspace_config.md#repository)) is no different: its hooks run around its own clone, start and stop, like any other project's.

### The workspace's `init.post`

The [workspace config](../configuration/workspace_config.md) has no extensions, and only one hook, `init.post`, which runs once every project is set up. Each entry names the project whose extension it runs:

```yaml
# orca.workspace.yaml
hooks:
  init:
    post:
      - extension: seed
        project: local-env
```

| Property | Description |
| -------- | ----------- |
| `extension` | The name of one of the project's extensions. |
| `project` | The project that defines it. The extension runs in that project's directory. |

### Which hooks there are

| Hook | Defined in | Runs |
| ---- | ---------- | ---- |
| `clone.post` | a project | After the project is cloned, in its directory, by `ws init` and `ws clone`. Again whenever they're re-run. |
| `up.pre`, `up.post` | a project | Before and after the project is started, by `orca up`. See [the order](./environments.md#hooks). |
| `down.pre`, `down.post` | a project | Before and after the project is stopped, by `orca down`. |
| `init.post` | the workspace config | At the end of `ws init`, after every project is cloned, and their `clone.post` hooks have run. |

There's no `init.pre` or `clone.pre`. A project's config can't be read until it's cloned, so nothing it defines can run before then.

A [library](./workspaces.md#library-projects) is never started or stopped, so it can only have `clone.post`.

## Running a hook by hand

[`orca hooks run`](../CLI/orca_hooks_run.md) runs every extension in a hook, in order:

```sh
$ orca hooks run clone.post -p api     # set api up again
$ orca hooks run init.post             # run the workspace's init.post, from anywhere
```

```
api clone.post: env
  Created .env from .env.example.
api clone.post: install-deps
  added 312 packages in 4s
```

- For `init.post`, it runs the workspace's, from anywhere, each extension in its own project. Every other hook is a project's, so it needs one: run it from inside the project, or pass `-p`.
- Each step's output is shown as it runs, indented under it.
- Before anything runs, any extension that's new, or has changed since you trusted it, is listed, and you're asked to trust it. See [Trusting extensions](./extensions.md#trusting-extensions). `--yes` runs everything without asking, for that run only.
- With [shell completion](./shell_completion.md) installed, `orca hooks run <tab>` lists the project's hooks that run something, and the workspace's `init.post`, with the extensions each one runs.

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
