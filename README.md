# Smallweb - Host websites from your internet folder

Smallweb is a lightweight web server based on [Deno](https://deno.com). It is inspired both by legacy specifications like [CGI](https://en.wikipedia.org/wiki/Common_Gateway_Interface) and online platforms like [Val Town](https://val.town) and [Deno Deploy](https://deno.com/deploy).

## A self-hosted serveless platform

Smallweb maps each folder in `~/www` folder to an unique domain. Ex: `~/www/example` will be mapped to:

- `https://example.localhost` on your local device
- `https://example.<your-domain>` on your homelab / VPS

Creating a new website becomes as simple a creating a folder and opening the corresponding url in your browser. No need to configure a build step, or start a development server. Since servers are mapped to folders, you can manage them using standard unix tools like `cp`, `mv` or `rm`.

Each incoming http request is sandboxed in a single deno subprocess by the smallweb evaluation server. If there is no activity on your website, no resources will be used, making it a great solution for low-traffic websites.

## Installation

All the instructions are written in the [getting started guide](https://pomdtr.github.io/smallweb).

## Demo

The following snippet is stored at `~/www/sqlite-example/main.ts` on my raspberrypi 400, and served at <https://sqlite-example.pomdtr.me>.

Any edit to the file is reflected in real-time, without the need to rebuild the project, or restart the server.

```tsx
// In smallweb, you install applications by just importing them
import { serveDatabase } from "jsr:@pomdtr/sqlite-explorer@0.4.0/server";
const handler = serveDatabase({ dbPath: "./chinook.db" });

// You can extends the functionality of your application by adding middlewares
import { lastlogin } from "jsr:@pomdtr/lastlogin@0.0.3";
const auth = lastlogin({
  // accept any email as valid.
  // In a real application, you would either whitelist emails,
  // or check the email against a database of users.
  verifyEmail: (_email) => true,
});

// This is the final handler that will be executed on each request
export default {
  fetch: auth(handler),
};
```
