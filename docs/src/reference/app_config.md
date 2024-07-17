# Configuration Reference

The smallweb configuration can be defined in a `smallweb.json[c]` file at the root of the project, or in the `smallweb` field of the `deno.json[c]` file. This config file is optional, and sensible defaults are used when it is not present.

A json schema for the config file is available [here](https://schema.smallweb.run).

You can set it using a `$schema` field in your `smallweb.json[c]` file:

```json
{
  "$schema": "https://schema.smallweb.run",
  "serve": "./build"
}
```

VS Code Users can also set it globally in their `settings.json`:

```json
{
  "json.schemas": [
    {
      "url": "https://schema.smallweb.run",
      "fileMatch": [
        "smallweb.json",
        "smallweb.jsonc"
      ]
    }
  ]
}
```

## `serve`

The `serve` field defines the file / directory to serve. If this field is not provided, the following defaults are used:

- if a `main.[js,jsx,ts,tsx]` file exists in the root directory, it is served
- if a `dist` directory exists and contains an `index.html` file, it is served
- the root directory is served

### Examples {#serve-examples}

- Serve a file

    ```json
    {
      "serve": "./serve.js"
    }
    ```

- Statically serve a directory

    ```json
    {
      "serve": "./build"
    }
    ```

## `crons`

The `crons` field defines a list of cron jobs to run. See the [Cron Jobs](../guides/cron.md) guide for more information.

```json
{
  "crons": [
    {
      "name": "example",
      "schedule": "0 0 * * *",
      "command": "./sync-db.ts",
    }
  ]
}
```

## `permissions`

The `permissions` field defines a list of permissions to grant to the Deno process. See the [Permissions](../guides/permissions.md) guide for more information.

### Examples {#permissions-examples}

- Giving all permissions

    ```json
    {
      "permissions": {
        "all": true
      }
    }
    ```

- Giving specific permissions

    ```json
    {
      "permissions": {
        "read": true,
        "write": {
          "allow": ["./", "/tmp"],
          "deny": ["./smallweb.json"]
        },
        "net": ["deno.land"]
      }
    }
    ```
