# Configuration Reference

The smallweb configuration can be defined in a `smallweb.json[c]` file at the root of the project. This config file is optional, and sensible defaults are used when it is not present.

A json schema for the config file is available [here](https://schema.smallweb.run).

You can set it using a `$schema` field in your `smallweb.json[c]` file:

```json
{
  "$schema": "https://schema.smallweb.run",
}
```

VS Code users can also set it globally in their `settings.json`:

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

## Available Fields

### `entrypoint`

The `entrypoint` field defines the file to serve. If this field is not provided, the app will look for a `main.[js,ts,jsx,tsx]` file in the root directory.

### `root`

The `root` field defines the root directory of the app. If this field is not provided, the app will use the app directory as the root directory.

### `private`

If the `private` field is set to `true`, the app will ask for your admin username and password before serving the app using basic auth.

### `crons`

The `crons` field defines a list of cron jobs to run. See the [Cron Jobs](../guides/cron.md) guide for more information.

```json
{
  "crons": [
    {
      "name": "daily-task", // The name of the cron task (required)
      "description": "A daily task", // A description for the task (optional)
      "schedule": "0 0 * * *", // a cron expression (required)
      "args": [] // arguments to pass to the task (required)
    }
  ]
}
```
