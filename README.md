# Smallweb - Host websites from your internet folder

Smallweb is a lightweight web server based on [Deno](https://deno.com). It draws inspiration from both legacy specifications like [CGI](https://en.wikipedia.org/wiki/Common_Gateway_Interface) and modern serverless platforms such as [Val Town](https://val.town) and [Deno Deploy](https://deno.com/deploy).

Smallweb maps domains to folders in your filesystem. For example, if you own the `pomdtr.me` domain:

- `https://www.pomdtr.me` maps to `~/smallweb/www`
- `https://example.pomdtr.me` maps to `~/smallweb/example`

Creating a new website is as simple as creating a folder and opening the corresponding URL in your browser. There's no need to configure a build step (unless you want to) or start a development server. Since servers are mapped to folders, you can manage them using standard Unix tools like `cp`, `mv`, or `rm`.

## A self-hosted serverless platform

Each incoming HTTP request is sandboxed in a single Deno subprocess by the Smallweb evaluation server. If there are no incoming requests, no resources are used, making it an ideal solution for low-traffic websites.

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
- <https://tldraw.smallweb.run>

Since creating smallweb websites is so easy, you can even create super simple ones. For example, when I want to invite someone to the smallweb discord server, I just send him the link <https://discord.smallweb.run>, which maps to `~/smallweb/discord/main.ts` on my vps.

```ts
export default {
    fetch: () => Response.redirect("https://discord.gg/BsgQK42qZe"),
};
```
