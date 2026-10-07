# Hosts & TLS

Orca can make your projects reachable at real-looking hostnames like `https://api.acme.test`, with certificates your browser trusts.

## Hosts

List each project's hostnames in its [`orca.project.yaml`](../configuration/project_config.md#hosts):

```yaml
hosts:
  - api.acme.test
```

Then add them to `/etc/hosts`:

```sh
$ sudo ORCA_CONFIG_PATH="$HOME/.orca/orca.yaml" orca hosts
```

[`orca hosts`](../CLI/orca_hosts.md) collects the hosts from every project in the workspace, and points each one at `127.0.0.1`. Its entries are tagged with a comment naming the workspace, and running it again replaces that workspace's entries, so it's safe to re-run after adding or removing hosts. Entries for other workspaces, and anything else in the file, are left alone.

Writing to `/etc/hosts` needs root, hence `sudo`.

### Why `ORCA_CONFIG_PATH` is set

Orca finds your [user config](../configuration/user_config.md) through your home directory (`$HOME/.orca/orca.yaml`), and that's where your workspaces are recorded. Whether `sudo` keeps your `HOME` depends on how it's configured:

- **Some systems reset `HOME` to root's home**, e.g. Ubuntu by default, or anywhere `sudoers` sets `always_set_home`. A plain `sudo orca hosts` then looks for root's config instead of yours. It doesn't find one, so it creates an empty, root-owned config at `/root/.orca/orca.yaml`, and fails because your workspace isn't in it.
- **Others keep your `HOME`**, which macOS's default `sudoers` does, as far as we know. There, plain `sudo orca hosts` finds your config.

Setting `ORCA_CONFIG_PATH` on the command line points orca at your config either way, so the same command works everywhere. If you know your `sudo` keeps `HOME`, you can leave it out.

!!! warning "Plugins run as root under `sudo`"
    Orca starts its [plugins](../plugins/index.md) on every run, so under `sudo` they run as root too:

    - plugins in `~/.orca/plugins`, when `sudo` keeps your `HOME`;
    - plugins in any `plugins.dirs` directories in your config, always, because `ORCA_CONFIG_PATH` points orca at that config.

    Setting `ORCA_PLUGINS_PATH=/nonexistent` skips the first, but not the second. Only run `orca hosts` with `sudo` if you trust every plugin you have installed.

## TLS certificates

List the domains each project needs certificates for in its [`orca.project.yaml`](../configuration/project_config.md#tls-certificates). Wildcards are allowed:

```yaml
tlsCerts:
  - "*.acme.test"
```

Then generate them:

```sh
$ orca tls gen
```

[`orca tls gen`](../CLI/orca_tls_gen.md) does two things:

1. **Creates a certificate authority** for orca, if there isn't one yet: `~/.orca/tls/cert.pem` and `key.pem`. It's shared by all your workspaces.
2. **Creates a certificate for each domain** in the workspace, signed by that authority, in `~/.orca/tls/certs/`. Each one is a `{domain}.cert` and `{domain}.key` pair, with `*` replaced by `_`. So `*.acme.test` becomes `_.acme.test.cert` and `_.acme.test.key`.

Existing certificates are kept, so it's safe to re-run after adding domains. Certificates are valid for two years. To replace one, delete its `.cert` and `.key` files and run `orca tls gen` again.

To use the certificates in a container, for example a reverse proxy, add the [TLS injection](../compose_files/overlays/TLS-injection.md) label to its service. Orca then mounts the certificates directory into it.

### Trusting the certificate authority

Browsers and other tools only accept the certificates once they trust orca's certificate authority. Orca doesn't do this for you, so add `~/.orca/tls/cert.pem` to your system's trust store once:

=== "macOS"
    ```sh
    $ sudo security add-trusted-cert -d -r trustRoot \
        -k /Library/Keychains/System.keychain ~/.orca/tls/cert.pem
    ```
=== "Debian / Ubuntu"
    ```sh
    $ sudo cp ~/.orca/tls/cert.pem /usr/local/share/ca-certificates/orca.crt
    $ sudo update-ca-certificates
    ```

Firefox uses its own trust store by default: import `cert.pem` under *Settings → Privacy & Security → Certificates → View Certificates → Authorities*.

!!! warning
    Keep `~/.orca/tls/key.pem` private. Anyone with it can create certificates your machine will trust.
