
# Network & Aliases

The primary purpose of the network overlay is simple; it joins all of the different compose projects in a workspace to the same network so they can access eachother. This makes use of the network configuration in docker compose files. The network overlay is enabled in the `orca.workspace.yaml` configuration, by adding a section like below. 

```
overlays:
  network:
    enabled: true
    createIn: {my-project}
    disableAliases: false
    aliasPattern: "{{ .Service }}.{{ .Project }}.{{ .Workspace }}.local"
```

| Property | Affect | Possible Values | Default |
| -------- | ------ | --------------- | ------- |
| `enabled` | Turns on the network overlay | `bool(true|false)` | `false` |
| `createIn` | The name of a project also defined within the workspace configuration, in which to create the network. The network will be defined in this project, and each other project will reference it as an "external" network. | `string` | `nil` (will cause an error) | 
| `disableAliases` | Prevents the creation of extra aliases on the container. By default the network overlay will generate an alias for each container following the format `service.project.workspace.local` so that there is a clear and predictable DNS name that can be used from any service to access any other. | `bool(true|false)` | `false` | 
| `aliasPattern` | A go template that defines the url that should be used. The go template will receive 3 variables `Service`, `Project`, `Workspace`, which are the names of the service in docker compose, the project name, and workspace name respectively. This can be used to customise the actual URL. | `string` | `{{ .Service }}.{{ .Project }}.{{ .Workspace }}.local` |

The shared network is named `orca-ws-{workspace}`, e.g. `orca-ws-acme`, so each workspace has its own network and several workspaces can run at the same time. It's created by the `createIn` project, and every other project joins it as an external network. So the `createIn` project must be started first: make every other project [`require`](../../configuration/workspace_config.md#dependencies-between-projects) it, directly or indirectly. Every service in every project is attached to the network.

!!! note "Upgrading from older versions"
    Older versions of orca named the network `orca-ws` for every workspace. After upgrading, the next `orca up` creates the new `orca-ws-{workspace}` network, but the old one isn't removed. Once nothing is using it, remove it with `docker network rm orca-ws`. If a running environment was started before the upgrade, run `orca down` and `orca up` on it to move it onto the new network.

The `aliases` overlay is included as a sub-feature in the network overlay, as they are very tightly linked; without knowing the network we don't know where to create the aliases. This is enabled by default but can be disabled with the `disableAliases` property shown above.

