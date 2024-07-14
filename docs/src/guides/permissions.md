# Configuring permissions

By default, a smallweb app can:

- read and write the current directory
- access environment variables using `Deno.env.get`
- access the network with `fetch`

If you want to add more permissions to your app (or restrict it even further), you can either:

- add `smallweb.json` configuration file at the root of the folder
- add a `smallweb` field in your `deno.json`

A json schema for the config file is available [here](https://schema.smallweb.run). See the deno docs to learn the [available permissions](https://docs.deno.com/runtime/manual/basics/permissions).

Here is the default config when no `smallweb.json` file is present:

```json
{
  "$schema": "https://schema.smallweb.run",
  "permissions": {
    "env": true,
    "net": true,
    "read": ["."],
    "write": {
      "allow": ["."],
      "deny": [ "smallweb.json", "smallweb.jsonc", "deno.json", "deno.jsonc"]
    }
  }
}
```

If you want to add permissions to run a binary, you should start from it, then add the required permissions:

```json
{
  "$schema": "https://schema.smallweb.run",
  "permissions": {
    "env": true,
    "net": true,
    "read": ["."],
    "write": {
      "allow": ["."],
      "deny": [ "smallweb.json", "smallweb.jsonc", "deno.json", "deno.jsonc"]
    },
    "run": ["/opt/homebrew/bin/gh"],
  }
}
```

As a general rule, you should only add permissions that are required for your app to run. The more permissions you add, the more attack surface you expose to potential attackers. If you know what you are doing (or just don't care), you can allow all permissions by setting the `all` field to `true`.

```json
{
  "$schema": "https://schema.smallweb.run",
  "permissions": {
    "all": true
  }
}
```
