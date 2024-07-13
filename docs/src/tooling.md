# Tooling

## Visual Studio Code

You can add a JSON schema to your Visual Studio Code settings to get autocompletion and validation for your `smallweb.json` files.

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

## Additional Tooling

- [Tailscale](https://tailscale.com/) - A VPN to connect to all your smallweb instances.
- [Mutagen](https://mutagen.io/) - A tool to keep your local and remote files in sync.
- [Cloudflare Tunnel](https://www.cloudflare.com/products/tunnel/) - A way to expose your smallweb instance to the internet.
