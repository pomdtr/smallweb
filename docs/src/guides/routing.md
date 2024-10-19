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

The routing system maps domains to directories as follows:

- Direct subdomain mapping:
  - `api.example.com` → `~/smallweb/api`
  - `blog.example.com` → `~/smallweb/blog`

- Root domain behavior:
  - Requests to `example.com` automatically redirect to `www.example.com` if the `www` directory exists
  - If the `www` directory does not exist, a 404 error is returned
