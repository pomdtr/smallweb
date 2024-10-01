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

As a special case, the root domain `example.com` is automatically redirected to `www.example.com`, so it will be mapped to `~/smallweb/www`.

If you want to opt-out of this behavior, you can create a `~/smallweb/@` directory, which will be mapped to the root domain.

## Custom domains

In addition to your base domain, you can also map custom domains to apps from your global config.

```json
{

    "domain": "example.com",
    "dir": "~/smallweb",
    "customDomains": {
        "pomdtr.me": "pomdtr"
    }
}
```

In this example, `pomdtr.me` will be mapped to `~/smallweb/pomdtr`, meaning that the `pomdtr` app will be accessible both at:

- `https://pomdtr.example.com`
- `https://pomdtr.me`

You can also map wildcards to apps by using the `*` character.

```json
{
    "domain": "example.com",
    "dir": "~/smallweb",
    "customDomains": {
        "pomdtr.me": "pomdtr",
        "sandbox-*.pomdtr.me": "pomdtr"
    }
}
```
