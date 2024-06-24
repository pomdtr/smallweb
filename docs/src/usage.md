Every smallweb app is defined as a folder in the `~/www` directory. The folder name will be used as the subdomain of the website. Dependending on your configuration, the `~/www/demo` folder could be served at:

- `https://demo.localhost`
- `https://demo.<your-domain>` (ex: <https://demo.pomdtr.me>)

Depending on the contend of the folder, the app can define:

- a static website
- an http server
- a cli command
- a combination of the above

## Hosting an HTTP server

Often, you will want to create a dynamic website. For this, you can create a file called `main.[js,ts,jsx,tsx]` in the folder. This file should export a default object with a `fetch` method that takes a `Request` object as argument, and returns a `Response` object.

Here is an example of a simple HTTP server:

```ts
// File: ~/www/demo/main.ts

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

Smallweb use the [deno](https://deno.com) runtime to evaluate the server code. You get typescript and jsx support out of the box, and you can import any module from the npm and jsr registry by prefixing the module name with `npm:` or `jsr:`.

As an example, the following code snippet use the `@hono/hono` extract params from the request url, and render jsx:

```tsx
// File: ~/www/hono/main.tsx
/** @jsxImportSource jsr:@hono/hono/jsx **/

import { Hono } from "@hono/hono";

const app = new Hono();

app.get("/", c => c.html(<h1>Hello, world!</h1>));

app.get("/:name", c => c.html(<h1>Hello, {c.req.param("name")}!</h1>));

export default app;
```

No need to run an install command, or configure typescript. Just copy-paste the snippet at `~/www/hono/main.tsx`, and open the corresponding url in your browser.

You are not limited to serving html responses. Smallweb is perfect for creating APIs, or small services (ex: discord/telegram bots, webhooks, etc)...

## Creating a static website

If the folder does not contains a `main.[js,ts,jsx,tsx]` file, Smallweb will serve the folder as a static website. The `index.html` file will be served as the root of the website.

Here is an example of a simple static website:

```html
<!-- File: ~/www/demo/index.html -->
<!DOCTYPE html>

My first <strong>Smallweb</strong> website!
```

A lot of static websites are distributed as github repositories. You can easily clone a repository in the `~/www` folder to create a new website. For example, you can clone the [sqlime](https://github.com/nalgeon/sqlime) repository to self-host your own sqlite playground:

```sh
git clone https://github.com/nalgeon/sqlime ~/www/sqlime
```

## Registering a CLI command

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

And if you want the command to be available globally, you can use the `smallweb install` command:

```sh
$ smallweb install demo
$ demo --name smallweb
Hello, smallweb!
```

Of course, you can define both an `main.ts` and a `cli.ts` file in the same folder.

## Setting env variables

You can set environment variables for your app by creating a file called `.env` in the folder.

Here is an example of a `.env` file:

```env
NAME=world
```

You can access the environment variables in your app using the `Deno.env` object:

```ts
// File: ~/www/demo/main.ts
export default function () {
  const name = Deno.env.get("NAME") || "world";

  return new Response(`Hello, ${name}!`, {
    headers: {
      "Content-Type": "text/plain",
    },
  });
}
```
