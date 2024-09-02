# Global Config

The smallweb config is located at `~/.config/smallweb/config.json[c]`. It is a json file that defines global settings for smallweb.

You can also specify a custom config file by setting the `SMALLWEB_CONFIG` environment variable.

Smallweb also respects the `XDG_CONFIG_HOME` environment variable.

## Available Fields

### `host`

The `host` field defines the host to bind to. By default, it is `127.0.0.1`.

```json
{
  "host": "0.0.0.0"
}
```

### `port`

The `port` field defines the port to bind to. By default, it is `7777`.

```json
{
  "port": 8000
}
```

### `domain`

The `domain` field defines the apex domain used for routing.

```json
{
  "domain": "smallweb.run"
}
```

See the [Routing](../guides/routing.md) guide for more information.

### `dir`

The `dir` field defines the root directory for all apps.

```json
{
  "dir": "~/smallweb"
}
```

### `env`

The `env` field defines a list of environment variables to set for all apps.

```json
{
  "env": {
    "NODE_ENV": "production"
  }
}
```

### `tokens`

The `tokens` field defines a list of tokens used for authentication.

```json
{
  "tokens": ["SF7RZt9shD6UnUcl"]
}
```

You can protect private apps by setting the `private` field in the app's config.

You can generate a new token using the `smallweb token` command (you'll still need to add it to your config).

Token are also used to protect internal services that smallweb provides, such as:

- webdav.`<domain>`: A webdav server allowing you to access your files.
- cli.`<domain>`: A web interface to run cli commands.

## Default Config

By default the config file looks like this:

```json
{
  "host": "127.0.0.1",
  "port": 7777,
  "domain": "localhost",
  "dir": "~/smallweb",
  "env": {
    // allow smallweb apps to communicate with each other when using self-signed certificates
    "DENO_TLS_CA_STORE": "system"
  }
}
```
