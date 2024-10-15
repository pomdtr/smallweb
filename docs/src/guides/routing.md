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

You can use the `X-Smallweb-App` header to route requests to a specific directory.

For example, if you are using caddy, you can use the following configuration:

```txt
example.com {
    header X-Smallweb-App "api"
    reverse_proxy localhost:7777
}
```
