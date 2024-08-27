# Global Config

The smallweb config is located at `~/.config/smallweb/config.json[c]`. It is a json file that defines global settings for smallweb.

You can also specify a custom config file by setting the `SMALLWEB_CONFIG` environment variable.

Smallweb also respects the `XDG_CONFIG_HOME` environment variable.

## `host`

The `host` field defines the host to bind to. By default, it is `127.0.0.1`.

```json
{
  "host": "0.0.0.0"
}
```

## `port`

The `port` field defines the port to bind to. By default, it is `7777`.

```json
{
  "port": 8000
}
```

## `domains`

The `domains` field defines a list of domains to folders. By default, it maps `*.localhost` to `~/smallweb/*`, but you can add more hostnames from the config.

```json
{
  "domains": {
    "example.com": "~/example.com"
  }
}
```

See the [Routing](../guides/routing.md) guide for more information.

## `env`

The `env` field defines a list of environment variables to set for all apps.

```json
{
  "env": {
    "NODE_ENV": "production"
  }
}
```
