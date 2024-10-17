# Routing

Smallweb maps every subdomains of your root domain to a directory in your root directory.

For example with, the following configuration:

```json
// ~/.config/smallweb/config.json
{
    "domain": "example.com",
    "dir": "~/smallweb"
}
```

Here, `api.example.com` will be mapped to `~/smallweb/api`, `blog.example.com` will be mapped to `~/smallweb/blog`, and so on.

The apex domain (example.com) will be mapped to `~/smallweb/@`.

If `~/smallweb/@` does not exist and `~/smallweb/www` does, every request to the apex domain (`example.com`) will be redirected to `www.example.com`. Inversely, if `~/smallweb/www` does not exist and `~/smallweb/@` does, every request to `www.example.com` will be redirected to the `example.com`.

If you want to opt-out of this behavior, you can create a `~/smallweb/@` directory, which will be mapped to the root domain.

## Custom domains

In addition to your base domain, you can also map custom domains to apps from your global config.

```json
{
    "domain": "example.com",
    "dir": "~/smallweb",
    "customDomains": {
        "pomdtr.me": "my-app"
    }
}
```

In this example, `pomdtr.me` will be mapped to `~/smallweb/my-app`, meaning that the `my-app` app will be accessible both at:

- `https://my-app.example.com`
- `https://pomdtr.me`
