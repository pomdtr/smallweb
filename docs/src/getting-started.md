# Getting started

## Why Smallweb ?

See <https://smallweb.run> for a quick overview of the project.

## Installation

If you want your apps to be available on the internet, you'll need to buy a domain name, and point it to your server.
You can find more information on to do this in the [Cloudflare Tunnel setup guide](./cloudflare/tunnel.md).

If you want to use a Virtual Private Server (VPS) to host your apps, you can follow the [VPS Setup](./vps.md). Hetzner Cloud, Digital Ocean, and Linode are good options for small projects.

If you prefer your to keep your apps local to your device, you can learn how to host your apps as `https://<app>.localhost` address in [this guide](./localhost/localhost.md). This option does not requires a domain name (or a server), but your app will only be available on your local device.

This guide will assumes that you have followed the [localhost setup guide](./localhost/localhost.md). If you haven't, just replace `https://<app>.localhost` with `https://<app>.<your-domain>` in the examples below.

### Hosting a static website

The simplest smallweb app you can create is just a folder with a text file in it.

```sh
mkdir -p ~/smallweb/localhost/example-website
echo "Hello, world!" > ~/smallweb/localhost/example-website/hello.txt
```

If you open `https://hello-world.localhost/hello.txt` in your browser, you should see the content of the file.

If the folder contains an `index.html` file (or a `dist/index.html` file), it will be served as the root of the website.

```html
<!-- File: ~/smallweb/localhost/example-website/index.html -->
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

If you want to serve dynamic content instead, you'll need to create a file called `main.[js,ts,jsx,tsx]` at the root of the folder. The file should export a default object with a `fetch` method that takes a `Request` object as argument, and returns a `Response` object.

```ts
// File: ~/smallweb/localhost/example-server/main.ts

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

```jsx
// File: ~/smallweb/localhost/hono-example/main.tsx
/** @jsxImportSource jsr:@hono/hono/jsx **/

import { Hono } from "@hono/hono";

const app = new Hono();

app.get("/", c => c.html(<h1>Hello, world!</h1>));

app.get("/:name", c => c.html(<h1>Hello, {c.req.param("name")}!</h1>));

export default app;
```

No need to start a development server, or to compile the code. Smallweb will take care of everything for you.

You can just copy paste this code at `~/smallweb/localhost/hono-example/main.tsx`, and open `https://hono-example.localhost` in your browser. The first load might take a few seconds, since deno is downloading the required modules, but subsequent loads will be instantaneous.

