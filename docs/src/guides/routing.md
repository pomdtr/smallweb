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

The apex domain (`example.com`) will be automatically redirected to `www.example.com`.

If you want to register a custom domain to a specific application, you can create a `CNAME` file in the application directory, with the custom domain name as the content of the file.
