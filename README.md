# Smallweb - Host websites from your internet folder

Smallweb is a lightweight web server based on [Deno](https://deno.com). It is
inspired both by legacy specifications like
[CGI](https://en.wikipedia.org/wiki/Common_Gateway_Interface) and serverless
platforms like [Val Town](https://val.town) and
[Deno Deploy](https://deno.com/deploy).

Smallweb maps domains to folder in your filesystem. For example, if you own the
`smallweb.run` domain:

- `https://smallweb.run` will be mapped to `~/smallweb/www`
- `https://example.smallweb.run` will be mapped to `~/smallweb/example`

Creating a new website becomes as simple a creating a folder and opening the
corresponding url in your browser. No need to configure a build step (unless you
want to), or start a development server. And since servers are mapped to
folders, you can manage them using standard unix tools like `cp`, `mv` or `rm`.

## A self-hosted serveless platform

Each incoming http request is sandboxed in a single deno subprocess by the
smallweb evaluation server. If there is no incoming request, no resources will
be used, making it a great solution for low-traffic websites.

Smallweb does not use Docker, but it still sandboxes your code using Deno. A smallweb app only has access to:

- the network
- some environment variables (for configuration or secrets)
- it own folder (read and write)

And if you website suddenly go viral, you can move your site to Deno Deploy in one command.

## Open-Source, GPL Licensed

You can find the smallweb source on [github](https://github.com/pomdtr/smallweb).

## Installation

All the instructions are available in the [docs](https://docs.smallweb.run).

## Examples

All the websites on the `smallweb.run` domain are hosted using smallweb (including this one):

- <https://docs.smallweb.run>
- <https://blog.smallweb.run>
- <https://api.smallweb.run>

Since creating smallweb websites is so easy, you can even create super simple ones. For example, when I want to invite someone to the smallweb discord server, I just send him the link <https://discord.smallweb.run>, which maps to `~/smallweb/discord/main.ts` on my vps.

```ts
export default {
    fetch: () => Response.redirect("https://discord.gg/BsgQK42qZe"),
};
```
