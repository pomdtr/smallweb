# Smallweb - Host websites from your internet folder

Smallweb is a lightweight web server based on [Deno](https://deno.com). It is inspired both by legacy specifications like [CGI](https://en.wikipedia.org/wiki/Common_Gateway_Interface) and online platforms like [Val Town](https://val.town) and [Deno Deploy](https://deno.com/deploy).

## A self-hosted serveless platform

Smallweb maps each folder in `~/www` folder to an unique domain. Ex: `~/www/example` will be mapped to:

- `https://example.localhost` on your local device
- `https://example.<your-domain>` on your homelab / VPS

Creating a new website becomes as simple a creating a folder and opening the corresponding url in your browser. No need to configure a build step (unless you want to), or start a development server. And since servers are mapped to folders, you can manage them using standard unix tools like `cp`, `mv` or `rm`.

Each incoming http request is sandboxed in a single deno subprocess by the smallweb evaluation server. If there is no incoming request, no resources will be used, making it a great solution for low-traffic websites. And if you website suddenly go virals, you can move your site to Deno Deploy in one command.

## Installation

All the instructions are written in the [getting started guide](https://smallweb-docs.pomdtr.me).

## Demo

The following snippet is stored at `~/www/sqlite-example/main.ts` on my raspberrypi 400.

```tsx
// In smallweb, you install applications by just importing them
import { serveDatabase } from "jsr:@pomdtr/sqlite-explorer@0.4.0/server";
// This is the final handler that will be executed on each request
export default { fetch: serveDatabase({ dbPath: "./chinook.db" }) };
```

These 2 lines of code provides me a full-featured LibSqlStudio UI at `https://sqlite-example.pomdtr.me`.

As a bonus, I'm also able to use the API endpoint to execute arbitrary SQL queries:

```sh
curl -X POST https://sqlite-example.pomdtr.me/api/execute -d '{ "statement": "SELECT Name FROM artists LIMIT 10" }'
```
