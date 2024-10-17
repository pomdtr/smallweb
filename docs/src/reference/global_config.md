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

### `email`

The `email` field is required to enable lastlogin authentication for private apps.

If it is not set, private will show a basic auth prompt instead.

```json
{
  "email": "pomdtr@smallweb.run"
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

### `customDomains`

The `customDomains` field defines a list of custom domains to map to apps.

```json
{
  "customDomains": {
    "pomdtr.me": "pomdtr",
  }
}
```

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
