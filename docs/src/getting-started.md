# Getting started

## Why Smallweb?

Smallweb maps each folder in `~/www` folder to an unique domain. Ex: `~/www/example` will be mapped to:

- `https://example.localhost` on your local device
- `https://example.<your-domain>` on your homelab / VPS

Creating a new website becomes as simple a creating a folder and opening the corresponding url in your browser. No need to configure a build step, or start a development server. And since servers are mapped to folders, you can manage them using standard unix tools like `cp`, `mv` or `rm`.

Every http request is sandboxed in a single deno subprocess by the smallweb evaluation server. If there is no activity on your website, no resources will be used on your server, making it a great solution for low-traffic websites.

## Installation

If you want your apps to be available on the internet, you'll need to buy a domain name, and point it to your server.
You can find more information on to do this in the [Cloudflare Tunnel setup guide](./cloudflare/tunnel.md).

If you prefer your to keep your apps local to your device, you can learn how to host your apps as `https://<app>.localhost` address in [this guide](./localhost/localhost.md). This option does not requires a domain name (or a server), but your app will only be available on your local device.

This guide will assumes that you have followed the [localhost setup guide](./localhost/localhost.md). If you haven't, just replace `https://<app>.localhost` with `https://<app>.<your-domain>` in the examples below.

### Hosting a web app

Every folder in the `~/www` directory is served statically by default. If the folder contains an `index.html` file, it will be served as the default page. Otherwise, the folder content will be listed.

To create a new website, just create a folder in the `~/www` directory, and add an `index.html` file in it.

```html
<!-- File: ~/www/example-website/index.html -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Smallweb - Host websites from your internet folder</title>
  <link href="https://cdnjs.cloudflare.com/ajax/libs/tailwindcss/2.2.19/tailwind.min.css" rel="stylesheet">
</head>
<body class="bg-white flex items-center justify-center min-h-screen text-black">
  <div class="border-4 border-black p-10 text-center">
    <h1 class="text-6xl font-extrabold mb-4">Smallweb</h1>
    <p class="text-2xl mb-6">Host websites from your internet folder</p>
  </div>
</body>
</html>
```

To access the website, open `https://example-website.localhost` in your browser.

If you want to serve dynamic content instead, you'll need to create a file called `main.[js,ts,jsx,tsx]` at the root of the folder. The file should export a default object with a `fetch` method that takes a `Request` object as argument, and returns a `Response` object.

```ts
// File: ~/www/example-server/main.ts

export default {
  fetch(request: Request) {
    const url = new URL(request.url);
    const name = url.searchParams.get("name") || "world";

    return new Response(`Hello, ${name}!`, {
      headers: {
        "Content-Type": "text/plain",
      },
    });
  },
}
```

To access the server, open `https://example-server.localhost` in your browser.

Smallweb use the [deno](https://deno.com) runtime to evaluate the server code. You get typescript and jsx support out of the box, and you can import any module from the npm and jsr registry by prefixing the module name with `npm:` or `jsr:`.

As an example, the following code snippet use the `@hono/hono` extract params from the request url, and render jsx:

```tsx
// File: ~/www/hono-example/main.tsx
/** @jsxImportSource jsr:@hono/hono/jsx **/

import { Hono } from "@hono/hono";

const app = new Hono();

app.get("/", c => c.html(<h1>Hello, world!</h1>));

app.get("/:name", c => c.html(<h1>Hello, {c.req.param("name")}!</h1>));

export default app;
```

No need to start a development server, or to compile the code. Smallweb will take care of everything for you.

You can just copy paste this code at `~/www/hono-example/main.tsx`, and open `https://hono-example.localhost` in your browser. The first load might take a few seconds, since deno is downloading the required modules, but subsequent loads will be instantaneous.

### Registering a CLI command

To add a cli command to your app, just create a file called `cli.[js,ts,jsx,tsx]` in the folder.

Here is an example of a simple cli command:

```ts
// File: ~/www/demo/cli.ts
import { parseArgs } from "jsr:@std/cli/parse-args";

const flags = parseArgs(Deno.args, {
  string: ["name"],
});

console.log(`Hello, ${flags.name || "world"}!`);
```

To run the command, you can use the `smallweb run` command:

```sh
$ smallweb run demo --name smallweb
Hello, smallweb!
```

Of course, you can define both an `main.ts` and a `cli.ts` file in the same folder.

### Setting env variables

You can set environment variables for your app by creating a file called `.env` in the application folder.

Here is an example of a `.env` file:

```env
BEARER_TOKEN=SECURE_TOKEN
```

Use the `Deno.env.get` method to access the environment variables in your app:

```ts
// File: ~/www/demo/main.ts
export default function (req: Request) {
  if (req.headers.get("Authorization") !== `Bearer ${Deno.env.get("BEARER_TOKEN")}`) {
    return new Response("Unauthorized", { status: 401 });
  }

  return new Response(`I'm private!`, {
    headers: {
      "Content-Type": "text/plain",
    },
  });
}
```

If you want to set an environment variable for all your apps, you can create a `.env` file in the `~/www` directory.

## Next steps

If you've read this far, you have already learned the whole smallweb API. You can now start building your own apps, by looking at the examples below:

> TODO: Add examples
