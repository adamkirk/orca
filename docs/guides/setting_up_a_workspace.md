# Setting up a workspace

This guide builds a complete workspace for a fictional company, Acme, with three repositories:

| Repository | Contents |
| ---------- | -------- |
| `acme/local-env` | Shared infrastructure: a [Traefik](https://traefik.io/) reverse proxy and a Postgres database. It also holds the workspace config. |
| `acme/api` | A backend API, that uses the database. |
| `acme/web` | A frontend, that calls the API. |

At the end, anyone on the team can run four commands to get the whole environment running at `https://api.acme.test` and `https://web.acme.test`.

The first half is for whoever sets the workspace up; the second half, [Joining the workspace](#joining-the-workspace), is what everyone else does.

## 1. The infrastructure project

Start with `local-env`, the project everything else depends on.

=== "orca.workspace.yaml"
    ```yaml
    name: acme

    overlays:
      network:
        enabled: true
        # The shared network is created when this project starts
        createIn: local-env

    projects:
      - name: local-env
        repository:
          # This repository is a project, as well as holding the workspace config
          self: true

      - name: api
        repository:
          ssh: git@github.com:acme/api.git
        requires:
          - local-env

      - name: web
        repository:
          ssh: git@github.com:acme/web.git
        requires:
          - local-env
          - api
    ```
=== "orca.project.yaml"
    ```yaml
    composeFiles:
      primary: docker-compose.yml

    tlsCerts:
      - "*.acme.test"

    hosts:
      - traefik.acme.test
    ```
=== "docker-compose.yml"
    ```yaml
    services:
      traefik:
        image: traefik:v3.1
        command:
          - --providers.docker=true
          - --providers.docker.exposedByDefault=false
          # Only route over orca's shared network
          - --providers.docker.network=orca-ws-acme
          - --providers.file.filename=/etc/traefik/dynamic.yaml
          - --entrypoints.web.address=:80
          - --entrypoints.web.http.redirections.entrypoint.to=websecure
          - --entrypoints.websecure.address=:443
          - --api.dashboard=true
        ports:
          - "80:80"
          - "443:443"
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock:ro
          - ./traefik/dynamic.yaml:/etc/traefik/dynamic.yaml:ro
        labels:
          # Orca mounts the generated TLS certificates here
          orca.pantoptescloud.tls/inject-certs: "/certs/"
          traefik.enable: "true"
          traefik.http.routers.dashboard.rule: Host(`traefik.acme.test`)
          traefik.http.routers.dashboard.entrypoints: websecure
          traefik.http.routers.dashboard.tls: "true"
          traefik.http.routers.dashboard.service: api@internal

      postgres:
        image: postgres:16
        environment:
          POSTGRES_USER: acme
          POSTGRES_PASSWORD: acme
        volumes:
          - postgres:/var/lib/postgresql/data

    volumes:
      postgres:
    ```
=== "traefik/dynamic.yaml"
    ```yaml
    tls:
      certificates:
        # "*" is replaced with "_" in generated certificate file names
        - certFile: /certs/_.acme.test.cert
          keyFile: /certs/_.acme.test.key
    ```

A few things to note:

- **`self: true`** registers this repository as the `local-env` project, rather than cloning it.
- **`createIn: local-env`** makes `local-env` create the shared `orca-ws-acme` network (`orca-ws-{workspace}`). The other projects join it as an external network, so they must start after `local-env`, which is why they all `require` it.
- **The network overlay** gives every service a DNS name on that network, `{service}.{project}.{workspace}.local`. So the API can reach Postgres at `postgres.local-env.acme.local`, without either compose file mentioning the other.
- **The compose file is a normal compose file.** The network, aliases and certificate mount are all added by orca's [overlay](../compose_files/overlays/index.md), so it still works with plain `docker compose`.

## 2. The application projects

Each application repository gets its own `orca.project.yaml`.

=== "api/orca.project.yaml"
    ```yaml
    composeFiles:
      primary: docker-compose.yml

    envFiles:
      - path: .env

    provisioners:
      # Fresh clones get a .env based on the committed example
      - exampleFile:
          src: .env.example
          target: .env
          type: dotenv

    hosts:
      - api.acme.test

    extensions:
      - name: migrate
        service: app
        command: ./bin/migrate
        defaultArgs:
          - up
      - name: test
        service: app
        command: go test
        defaultArgs:
          - ./...
    ```
=== "api/docker-compose.yml"
    ```yaml
    services:
      app:
        build: .
        environment:
          DATABASE_URL: postgres://acme:acme@postgres.local-env.acme.local:5432/acme
          LOG_LEVEL: ${LOG_LEVEL:-info}
        labels:
          traefik.enable: "true"
          traefik.http.routers.api.rule: Host(`api.acme.test`)
          traefik.http.routers.api.entrypoints: websecure
          traefik.http.routers.api.tls: "true"
          traefik.http.services.api.loadbalancer.server.port: "8080"
    ```
=== "api/.env.example"
    ```sh
    LOG_LEVEL=debug
    ```
=== "web/orca.project.yaml"
    ```yaml
    composeFiles:
      primary: docker-compose.yml

    hosts:
      - web.acme.test

    extensions:
      - name: npm
        service: web
        command: npm
    ```
=== "web/docker-compose.yml"
    ```yaml
    services:
      web:
        build: .
        environment:
          API_URL: https://api.acme.test
        labels:
          traefik.enable: "true"
          traefik.http.routers.web.rule: Host(`web.acme.test`)
          traefik.http.routers.web.entrypoints: websecure
          traefik.http.routers.web.tls: "true"
          traefik.http.services.web.loadbalancer.server.port: "3000"
    ```

Add `.env` to each repository's `.gitignore`, commit everything, and push.

## Joining the workspace

Everyone else (and you, on a fresh machine) sets up the environment like this.

### Clone and register everything

```sh
$ mkdir -p ~/code/acme && cd ~/code/acme
$ git clone git@github.com:acme/local-env.git
$ cd local-env
$ orca ws init
```

[`orca ws init`](../CLI/orca_ws_init.md) reads `orca.workspace.yaml` from the current directory and registers the `acme` workspace in your [user config](../configuration/user_config.md). It then sets up each project:

- `local-env` is registered where it is, because it's `self: true`.
- `api` and `web` are cloned next to it, into `~/code/acme/api` and `~/code/acme/web`. Each project is cloned into a directory named after the project.

Repositories you've already cloned in those locations are detected (by their git remote) and registered, rather than cloned again.

??? tip "Cloning somewhere else"
    By default projects are cloned into the parent of the directory you run `ws init` from. Use `--target` to choose another directory, e.g. `orca ws init --target ~/src`.

Make it your current workspace, so commands work from anywhere:

```sh
$ orca ws switch acme
$ orca ws ls
```

### One-off setup

Create each project's `.env` from its example:

```sh
$ (cd ~/code/acme/api && orca provision)
```

Generate TLS certificates, and add the hosts to `/etc/hosts`:

```sh
$ orca tls gen
$ sudo ORCA_CONFIG_PATH="$HOME/.orca/orca.yaml" orca hosts
```

`orca hosts` needs `sudo` to write to `/etc/hosts`. `ORCA_CONFIG_PATH` makes sure it reads your config rather than root's, as some systems change `HOME` under `sudo`. See [why `ORCA_CONFIG_PATH` is set](../usage/hosts_and_tls.md#why-orca_config_path-is-set) for the details, and a note about plugins.

Then trust orca's certificate authority, so your browser accepts the certificates. See [Hosts & TLS](../usage/hosts_and_tls.md#trusting-the-certificate-authority) for how.

### Start it

```sh
$ orca up -w acme
```

Orca starts `local-env` first, then `api`, then `web`, following `requires`.

!!! tip
    `-w acme` matters here: we're still inside the `local-env` directory, so a plain `orca up` would only start `local-env`. From outside any project directory, `orca up` starts the whole current workspace. See [Choosing a workspace and project](../getting_started/context.md). Once they're up, `https://web.acme.test` and `https://api.acme.test` should load in your browser.

## Day to day

```sh
$ cd ~/code/acme/api
$ orca logs                 # follow the API's logs
$ orca ext migrate          # runs "./bin/migrate up" in the app service
$ orca ext migrate down 1   # runs "./bin/migrate down 1" instead
$ orca ext test             # runs the tests
$ orca exec -s app sh       # a shell in the app container
$ orca restart              # restart just the API

$ orca g status --all       # current branch and changes, in every project
$ orca g pull --all         # pull the current branch in every project

$ orca down -w acme         # stop everything
```

See [Using orca](../usage/index.md) for everything else you can do.

## Variations

### A separate repository for the workspace

You don't need a `self` project. If the workspace config lives in a repository with nothing to run (say, `acme/environment`), just leave it out of `projects`, and make one of the cloned projects the `createIn` project instead.

### Adding a project later

Add it to `orca.workspace.yaml` and commit it. Everyone else then pulls the change and runs:

```sh
$ orca ws clone -p new-project
```

### A single repository

A workspace can have just one project. A repository with its own `orca.workspace.yaml` (with one `self: true` project) and `orca.project.yaml` still gets you hosts, TLS, extensions and the rest. Leave the network overlay disabled, as there's nothing to connect it to.

### Two workspaces at once

Each workspace gets its own network, `orca-ws-{workspace}`, so you can run several workspaces at the same time. Services in different workspaces can't reach each other over these networks. Only one of them can publish a given host port, though, so if two workspaces each run a proxy on ports 80 and 443, start just one of those at a time.
