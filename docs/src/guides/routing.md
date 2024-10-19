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

- `api.example.com` is mapped to the directory `~/smallweb/api`
- `blog.example.com` is mapped to `~/smallweb/blog`
- Any subdomains following the pattern `*.<app>.example.com` (e.g., `sub.api.example.com`) will be handled by `~/smallweb/<app>` (e.g., `~/smallweb/api`).

For the apex domain (`example.com`), it will map to the directory `~/smallweb/@`.

If the directory `~/smallweb/@` does not exist but `~/smallweb/www` does, all requests to the apex domain (example.com) will be redirected to `www.example.com`.

Conversely, if `~/smallweb/www` does not exist but `~/smallweb/@` does, requests to `www.example.com will` be redirected to example.com.
