# Smallweb - Host websites from your internet folder

Smallweb is a lightweight web server based on [Deno](https://deno.com). It is inspired both by legacy specifications like [CGI](https://en.wikipedia.org/wiki/Common_Gateway_Interface) and online platforms like [Val Town](https://val.town) and [Deno Deploy](https://deno.com/deploy).

Smallweb maps hostnames to folder in your home directory:

- `example.localhost` will be mapped to "~/smallweb/example.localhost"
- `example.smallweb.run` will be mapped to "~/smallweb/example.smallweb.run"

Creating a new website becomes as simple a creating a folder and opening the corresponding url in your browser. No need to configure a build step (unless you want to), or start a development server. And since servers are mapped to folders, you can manage them using standard unix tools like `cp`, `mv` or `rm`.

## A self-hosted serveless platform

Each incoming http request is sandboxed in a single deno subprocess by the smallweb evaluation server. If there is no incoming request, no resources will be used, making it a great solution for low-traffic websites.

And if you website suddenly go viral, you can move your site to Deno Deploy in one command.

## Installation

All the instructions are written in the [getting started guide](https://docs.smallweb.run).

## Demo

The following snippet is stored at `$SMALLWEB_ROOT/smallweb.run/sqlite-example/main.ts` on my raspberrypi 400.

```tsx
// In smallweb, you install applications by just importing them
import { serveDatabase } from "jsr:@pomdtr/sqlite-explorer@0.4.0/server";

// This is the final handler that will be executed on each request
export default { fetch: serveDatabase({ dbPath: "./chinook.db" }) };
```

These 2 lines of code provides me a full-featured LibSqlStudio UI that you can access at <https://sqlite-example.smallweb.run>.

As a bonus, I'm also able to use the API endpoint to execute arbitrary SQL queries:

```sh
curl https://sqlite-example.smallweb.run/api/execute \
    -X POST \
    -d '{ "statement": "SELECT Name FROM artists LIMIT 10" }'
```
