# Global Config

The smallweb config is located at `~/.config/smallweb/config.json[c]`. It is a json file that defines global settings for smallweb.

You can also specify a custom config file by setting the `SMALLWEB_CONFIG` environment variable.

Smallweb also respects the `XDG_CONFIG_HOME` environment variable.

## Available Fields

### `addr`

The `addr` field defines the addr to bind to. By default, it is `:7777`.

```json
{
  "addr": "127.0.0.1:8000"
}
```

If you want to use an unix socket, you can use the `unix/` prefix.

```json
{
  "addr": "unix/~/smallweb.sock"
}
```

### `cert` and `key`

The `cert` and `key` fields define the path to the SSL certificate and key.

```json
{
  "cert": "~/cert.pem",
  "key": "~/key.pem"
}
```

### `domain`

The `domain` field defines the apex domain used for routing.

```json
{
  "domain": "example.com"
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
  "email": "pomdtr@example.com"
}
```

### `customDomains`

The `customDomains` field is an object that maps custom domains to apps.

```json
{
  "customDomains": {
    "example.com": "example",
  }
}
```

## Default Config

By default the config file looks like this:

```json
{
  "addr": ":7777",
  "dir": "~/smallweb",
}
```

Since smallweb requires a domain to be set, the minimal config is:

```json
{
  "domain": "example.com"
}
```

which is equivalent to:

```json
{
  "domain": "example.com",
  "addr": ":7777",
  "dir": "~/smallweb",
}
```
