# Release process

The release process is fully automated for this project. It's centered around conventional commits and semantic versioning. We use a couple of tools to help with this:

- [semantic-release](https://semantic-release.gitbook.io/semantic-release/)
- [Goreleaser](https://goreleaser.com/)

The full process for shipping a new version of the application is shown below. The main thing to remember is that we must use conventional commits correctly, as the `type` of the commit largely determines the next version for release. For example, if we create a `fix` commit then the patch version will be incremented, however if we use a `feat` commit then the minor version will be incremented.

*See [Github Actions](./github_actions.md) for more info on the exact steps.*

## Versioning before 1.0

Orca is still on `0.x.x` versions, and follows the same approach as [Terraform](https://github.com/hashicorp/terraform/blob/main/README.md) did before its 1.0 release: until 1.0, orca is considered unstable, and **any release can include changes that would be breaking under semantic versioning**, such as removing a command, renaming a flag or changing the config format. The commit types still decide the version as usual, so features bump the minor version, and fixes and most other types bump the patch version.

What that means in practice:

- **Never mark a commit as breaking.** Don't use `!` after the type (`feat!:`) or a `BREAKING CHANGE:` footer. semantic-release turns a breaking commit into a major release, which on `0.x` would release `1.0.0`.
- **Describe changes in behaviour in the commit body instead.** Use the normal type (`feat`, `fix`, `refactor`...). In the body, say what changed for users, and what they need to do about it, e.g. a removed command and what to use instead. That text ends up in the release notes.
- **Document the change** in the user docs as part of the same PR, as with any other change in behaviour.

### The 1.0 release

The maintainer decides when orca reaches 1.0, and makes that release deliberately. Nothing else should cause it. From 1.0 onwards, orca follows semantic versioning properly: breaking changes are marked as breaking, and bump the major version.

??? tip "Workflow"
    ```mermaid
    flowchart TD
        A([Dev]) -->|Creates feature branch| B(Makes changes)
        B --> |Creates PR| C{Quality checks}
        C -->|Approved & CI Passed| D[Merges PR to main]
        C -->|Changes Requested OR CI Failed| B
        D -->|Release process starts| E[semantic-release: Calculates next version based on Conventional Commits]
        E -->|Result| F{Requires release}
        F -->|Yes| G[New Tag created]
        F -->|No| H([No release created])
        G --> I([Goreleaser: publishes new release for tag])
    ```

## semantic-release

The semantic release configuration can be found in `.releaserc`, and it's pretty minimal. The current configuration is setup to only create a tag in the git repo, and not to create a github release or changelog etc. This is by design, so that we can rely on goreleaser to generate a new release that includes the artifacts for downloading the executable.

## goreleaser

The goreleaser configuration is found in `.goreleaser.yaml`, and is mostly boilerplate. There are a few minor changes to tell it where to find the entrypoint of the application code, and make the release notes a little more organised.