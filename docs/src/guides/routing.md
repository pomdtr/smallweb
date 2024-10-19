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

In this setup:

- api.example.com is mapped to the directory ~/smallweb/api
- blog.example.com is mapped to ~/smallweb/blog
- Any subdomains following the pattern `*.<app>.example.com` (e.g., sub.api.example.com) will be handled by `~/smallweb/<app>` (e.g., ~/smallweb/api).

For the apex domain (example.com), it will map to the directory ~/smallweb/@.

If the directory `~/smallweb/@` does not exist but `~/smallweb/www` does, all requests to the apex domain (example.com) will be redirected to `www.example.com`.

Conversely, if `~/smallweb/www` does not exist but `~/smallweb/@` does, requests to `www.example.com will` be redirected to example.com.


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

Subdomains of custom domains will also be mapped to the same directory. For example, `sub.pomdtr.me` will be handled by `~/smallweb/my-app`.
