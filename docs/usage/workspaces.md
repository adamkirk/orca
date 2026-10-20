# Workspaces

The `orca ws` commands register workspaces on your machine, clone their projects, and choose which workspace is current. For a full example, see [Setting up a workspace](../guides/setting_up_a_workspace.md).

## Initialising a workspace

You can initialise a workspace straight from its GitHub repository, or from a copy of the repository you've already cloned.

### From GitHub

```sh
$ mkdir -p ~/code/acme && cd ~/code/acme
$ orca ws init --repo acme/local-env
```

With `--repo`, [`orca ws init`](../CLI/orca_ws_init.md):

1. Fetches the workspace config from the repository's default branch, or `--branch`, through the GitHub API, and checks that it's [valid](../configuration/workspace_config.md#validation). If the branch or the config is missing, or the config is invalid, orca stops before cloning anything.
2. Checks that every project's [`repository.branch`](../configuration/workspace_config.md#repository) exists on its remote. If any are missing, orca lists them all and stops before cloning anything.
3. Clones the repository over SSH (`git@github.com:acme/local-env.git`) into `{target}/{repo name}`, here `~/code/acme/local-env`, on `--branch` if it's given. If that directory is already a clone of the repository, e.g. as you're running `ws init` again, it's used as it is. If it's anything else, orca stops with an error rather than touch it.
4. Carries on as if you'd run `orca ws init` from inside the clone (see [below](#from-a-local-directory)), so the other projects are cloned next to it.

`--repo` only accepts GitHub repositories, in the form `owner/repo`. You'll need SSH access to the repository for the clone.

For a private repository, set `GH_TOKEN` or `GITHUB_TOKEN` to a token that can read it, as you would for the [GitHub CLI](https://cli.github.com/manual/gh_help_environment). `GH_TOKEN` is used if both are set. Without a token, a private repository looks the same as one without a workspace config.

### From a local directory

```sh
$ cd ~/code/acme/local-env      # a repository containing orca.workspace.yaml
$ orca ws init
```

Without `--repo`, [`orca ws init`](../CLI/orca_ws_init.md):

1. Reads the workspace config from the current directory, checks that it's [valid](../configuration/workspace_config.md#validation), and registers the workspace in your [user config](../configuration/user_config.md).
2. Creates your [preferences file](../configuration/preferences.md), listing every project, unless it already exists. Each project follows the workspace config's [`enabled`](../configuration/workspace_config.md#projects) until you enable or disable it.
3. Clones every project, as [`orca ws clone`](#cloning-projects) does.
4. Runs every project's `clone.post` [hook](./hooks.md), in the order they're cloned, then the workspace's `init.post`. Any extension they run that you haven't trusted yet is listed first, in one go, and you're asked to [trust](./extensions.md#trusting-extensions) it.
5. Lists what's left to do, which orca never does itself, e.g. as it needs `sudo`:

    ```
    Workspace acme is ready. Next steps:
      orca ws switch acme                                        make it your current workspace
      sudo ORCA_CONFIG_PATH="$HOME/.orca/orca.yaml" orca hosts   add its hosts to /etc/hosts
      orca tls gen                                               generate its TLS certificates
      orca up                                                    start it
    ```

    If git doesn't ignore the preferences file, orca also suggests adding it to `.gitignore`.

### Running it again

`orca ws init` is safe to run again, in the same place. If the workspace is already registered from the same config, it carries on: it clones any projects that are missing, then runs every hook again. Nothing records which hooks have run, so this is how to finish a setup that failed part-way, once you've fixed the problem. It only asks about extensions that are new, or have changed since you trusted them.

That relies on hooks being safe to run more than once, see [Write extensions that are safe to run again](./hooks.md#write-extensions-that-are-safe-to-run-again). To clone without running anything, pass `--skip-hooks`.

A different workspace with the same name is still an error.

### Flags

| Flag | Description | Default |
| ---- | ----------- | ------- |
| `-r`, `--repo` | The GitHub repository to initialise from, as `owner/repo`. Can't be used with `--source`. | |
| `-b`, `--branch` | The branch of the `--repo` repository to read the workspace config from, and to clone. It only applies to the workspace repository: each project's branch comes from its `repository.branch`. Can't be used with `--source`, which uses the local checkout on whichever branch it's on. | The repository's default branch |
| `-s`, `--source` | The directory containing the workspace config. | The current directory |
| `-c`, `--config` | The workspace config's file name, relative to the source directory or the repository's root. | `orca.workspace.yaml` |
| `-t`, `--target` | The directory to clone projects into. | The current directory with `--repo`, otherwise its parent |
| `--skip-hooks` | Clone without running any hooks. | |
| `-y`, `--yes` | Run every extension the hooks run without asking, for this run only: it doesn't record them as trusted. | |

It doesn't make the workspace current; use `orca ws switch` for that.

## Cloning projects

```sh
$ orca ws clone                 # every project in the workspace
$ orca ws clone -p payments     # just one project
```

[`orca ws clone`](../CLI/orca_ws_clone.md) sets up any projects that aren't set up yet. Run it after a project is added to the workspace config. For each project:

- A `self: true` project is registered at the root of the workspace config's repository.
- If its directory exists and the project is already registered, it's skipped.
- If its directory exists and is a clone of the project's repository (checked by its git remote), it's registered as is.
- If its directory exists but is something else, orca stops with an error, rather than overwrite it.
- Otherwise the repository is cloned, on its `repository.branch` if it has one, and registered.

Before cloning anything, orca checks that the `repository.branch` of every project it's about to clone exists on its remote, and lists every missing one. The branch is only used for the first clone: projects that are already cloned are never switched to another branch.

Projects are cloned into `{target}/{project name}`, or `{target}/{path}` for a project with a [`path`](../configuration/workspace_config.md#projects). `--target` defaults to the parent of the workspace config's repository, so projects end up next to it. With `-p` and `--target`, the project is cloned into `--target` itself.

Projects are cloned in the order they're configured, except that a project is cloned after the projects it [`requires`](../configuration/workspace_config.md#dependencies-between-projects), and after any project whose directory its `path` is inside.

[Disabled](#enabling-and-disabling-projects) projects are cloned too: being disabled only means they don't run.

Then the `clone.post` [hook](./hooks.md) of every project it was asked for runs, in the same order, whether it was cloned now or before. So running `orca ws clone` again sets the projects up again. `--skip-hooks` clones without running them, and `--yes` runs them without asking about untrusted extensions.

## Enabling and disabling projects

If you only work on part of a workspace, disable the projects you don't need, and `orca up` won't start them. Your choices are kept in your [preferences file](../configuration/preferences.md), and override the defaults in the workspace config's [`enabled`](../configuration/workspace_config.md#projects).

```
$ orca ws projects
┌───────────┬─────────┬─────────┬────────┬─────────┐
│  Project  │  Type   │ Enabled │ Cloned │ Running │
├───────────┼─────────┼─────────┼────────┼─────────┤
│ local-env │ service │ yes     │ yes    │ yes     │
│ api       │ service │ yes     │ yes    │ yes     │
│ web       │ service │ yes     │ yes    │ no      │
│ admin     │ service │ yes     │ yes    │ no      │
└───────────┴─────────┴─────────┴────────┴─────────┘

$ orca ws disable web admin
Disabled web and admin.

$ orca up                 # starts local-env and api
$ orca up -p web
Note: web is disabled, but starting it as you asked for it.
```

| Command | Does |
| ------- | ---- |
| [`orca ws projects`](../CLI/orca_ws_projects.md) | Lists the workspace's projects, with whether each is enabled, cloned and running. Running is `?` if docker can't be asked, rather than the command failing. |
| [`orca ws enable <project>...`](../CLI/orca_ws_enable.md) | Enables projects. It doesn't start them. |
| [`orca ws disable <project>...`](../CLI/orca_ws_disable.md) | Disables projects, and stops any of them that are running. With `--orphan`, they're left running. |

All of them take `-w`, and complete project names: `ws enable` offers the disabled projects, and `ws disable` the enabled ones.

### What disabled means

- **Workspace-wide commands skip disabled projects.** `orca up` only starts enabled projects, and [`g` with `--all`](./git.md) only acts on enabled projects.
- **An explicit `-p`** still acts on a disabled project. `orca up -p web` starts it, with a note that it's disabled.
- **Requirements win.** If an enabled project [`requires`](../configuration/workspace_config.md#dependencies-between-projects) a disabled one, `orca up` starts it anyway, and warns:

    ```
    Warning: api requires local-env, which is disabled, so it's being started anyway.
    ```

- **`orca down` stops everything that's running.** As well as what `orca up` starts, it stops any disabled project that's still running, so nothing is left behind.
- **Nothing is deleted.** Disabling a project never removes its clone or its containers.

### Disabling a running project

`orca ws disable` stops any of the projects that are running, as `orca down -p <project>` would:

```
$ orca ws disable web
Disabled web.
web:acme[/Users/me/code/acme/web] stopping...
```

- With `--orphan`, they're left running, and orca tells you how to stop them later.
- A project that another running project still requires is left running, with a warning, as requirements win. Disabling both at once stops both.

## Library projects

Some repositories don't have anything to run: shared packages, SDKs or config repositories that only need cloning into the right place. Give them `type: library` in the [workspace config](../configuration/workspace_config.md#projects), and orca clones and keeps them up to date with everything else, but never tries to run them:

```yaml
projects:
  - name: sdk
    type: library
    repository:
      ssh: git@github.com:acme/sdk.git
    path: api/packages/sdk
```

```
$ orca ws clone
Cloning 'git@github.com:acme/api.git' into /Users/me/code/acme/api...
Cloning 'git@github.com:acme/sdk.git' (library) into /Users/me/code/acme/api/packages/sdk...

$ orca up -p sdk
Error: sdk is a library project, so it has no compose stack for this command to act on!
```

- **Included:** `ws init`, `ws clone`, `ws projects`, `ws enable`/`disable`, and the [`g` commands](./git.md) (with and without `--all`).
- **Not included:** `up`, `down`, `restart`, `logs`, `exec`, `debug`, `hosts` and `tls gen`. Workspace-wide, they skip libraries. With `-p <library>`, or run inside a library's directory, they fail with the error above. `-p <tab>` doesn't offer libraries for them.
- **Inside a library's directory**, orca acts on the library, even when it's cloned inside another project, so `orca g status` there shows the library's status.
- **Project config is optional.** See [Project config](../configuration/project_config.md) for what a library's can contain.
- **Required libraries that aren't cloned:** if `orca up` starts a project that requires a library that isn't cloned yet, e.g. one added to the workspace config since you last cloned, it warns, and starts the project anyway:

    ```
    Warning: web requires sdk, which isn't cloned; clone it with 'orca ws clone -p sdk'.
    ```

- **Disabling** a library only leaves it out of `g --all`, as there's nothing to run.

### Cloning inside another project

A `path` inside another project puts the library inside that project's git repository. Unless that repository ignores it, the library shows up there as untracked files, so `orca ws clone` warns about it:

```
Warning: sdk is inside api, whose repository doesn't ignore packages/sdk, so it shows up there as untracked files. Add it to api's .gitignore.
```

The project it's inside is always cloned first. `orca ws clone -p sdk` fails if that project isn't cloned yet, and tells you to clone it first, as cloning the library would create its directory, which couldn't then be cloned into.

## Switching workspaces

| Command | Does |
| ------- | ---- |
| [`orca ws ls`](../CLI/orca_ws_ls.md) | Lists your workspaces, and which is current. |
| [`orca ws switch <name>`](../CLI/orca_ws_switch.md) | Makes a workspace the current one. |
| [`orca ws current`](../CLI/orca_ws_current.md) | Shows the current workspace. |
| [`orca ws clear-current`](../CLI/orca_ws_clear-current.md) | Unsets the current workspace. |

The current workspace is used when you're not inside any project's directory and don't pass `-w`. See [Choosing a workspace and project](../getting_started/context.md).
