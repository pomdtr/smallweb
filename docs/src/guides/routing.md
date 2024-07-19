# Routing

Smallweb maps domains to folders. By default, it will maps `*.localhost` to `~/localhost/*`, but you can add more hostnames from the config (`~/.config/smallweb/config.json[c]`).

## Mapping a single domain to a folder

```json
{
  "domains": {
    "example.com": "~/example.com"
  }
}
```

## Mapping a wildcard domain to a single folder

```json
{
  "domains": {
    "*.example.com": "~/example.com",
  }
}
```

## Mapping a wildcard domain to multiple folders

```json
{
  "domains": {
    "*.example.com": "~/example.com/*"
  }
}
```
