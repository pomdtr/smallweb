<div class="oranda-hide">

# Smallweb - Host websites from your internet folder

</div>

Smallweb is a lightweight web server based on [Deno](https://deno.com).

It is inspired both by [CGI](https://en.wikipedia.org/wiki/Common_Gateway_Interface) and modern serverless platfrom like [Val Town](https://val.town) and [Deno Deploy](https://deno.com/deploy).

Smallweb maps each folder in `~/www` to a subdomain (`~/www/example` -> `https://example.localhost` or `https://example.<my-domain>`). Smallweb is not limited to serving static files, it can also run server-side code, and interact with the file system.

Creating a new website becomes as simple as creating a new folder, or cloning a git repository. You can manage your servers using the standard file system tools, and deploy new versions by simply copying files.

You can test smallweb in a few minutes by following the [Getting Started](./getting-started.md) guide.
