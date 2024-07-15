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

## Using JSX

You can use the `@jsxImportSource` pragma to define the source of the jsx factory function. This allows you to use jsx in your server code.

```tsx
// File: ~/smallweb/localhost/jsx-example/main.tsx
/** @jsxImportSource npm:@preact **/
import render from "npm:preact-render-to-string";

const handler = () => new Response(render(<h1>Hello, world!</h1>), {
  headers: {
    "Content-Type": "text/html",
  },
});

export default { fetch: handler };
```

To access the server, open `https://jsx-example.localhost` in your browser.

## Routing Requests using Hono

Smallweb use the [deno](https://deno.com) runtime to evaluate the server code. You get typescript and jsx support out of the box, and you can import any module from the npm and jsr registry by prefixing the module name with `npm:` or `jsr:`.

As an example, the following code snippet use the `@hono/hono` extract params from the request url.

```jsx
// File: ~/smallweb/localhost/hono-example/main.ts

import { Hono } from "jsr:@hono/hono";

const app = new Hono();

app.get("/", c => c.text("Hello, world!"));

app.get("/:name", c => c.text(`Hello, ${c.params.name}!`));

export default app;
```

To access the server, open `https://hono-example.localhost` in your browser.
