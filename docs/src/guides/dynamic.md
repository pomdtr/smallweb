# Dynamic Websites

Smallweb can also host dynamic websites. To create a dynamic website, you need to create a folder with a `main.[js,ts,jsx,tsx]` file in it.

The file should export a default object with a `fetch` method that takes a `Request` object as argument, and returns a `Response` object.

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
