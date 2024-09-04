# Hosting Websites

Smallweb can also host dynamic websites. To create a dynamic website, you need to create a folder with a `main.[js,ts,jsx,tsx]` file in it.

The file should export a default object with a `fetch` method that takes a `Request` object as argument, and returns a `Response` object.

```ts
// File: ~/smallweb/example-server/main.ts

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
// File: ~/smallweb/jsx-example/main.tsx
/** @jsxImportSource npm:@preact **/
import render from "npm:preact-render-to-string";

const requestHandler = () => new Response(render(<h1>Hello, world!</h1>), {
  headers: {
    "Content-Type": "text/html",
  },
});

export default { fetch: requestHandler };
```

To access the server, open `https://jsx-example.localhost` in your browser.

## Routing Requests using Hono

Smallweb use the [deno](https://deno.com) runtime to evaluate the server code. You get typescript and jsx support out of the box, and you can import any module from the npm and jsr registry by prefixing the module name with `npm:` or `jsr:`.

As an example, the following code snippet use the `@hono/hono` extract params from the request url.

```jsx
// File: ~/smallweb/hono-example/main.ts

import { Hono } from "jsr:@hono/hono";

const app = new Hono();

app.get("/", c => c.text("Hello, world!"));

app.get("/:name", c => c.text(`Hello, ${c.params.name}!`));

// Hono instances have a `fetch`, so they can be used as the default export
export default app;
```

To access the server, open `https://hono-example.localhost` in your browser.

## Static Websites

You have multiple options to host static websitest with Smallweb.

First, you can store your static websites in a subdirectory, the server it using the `@std/http/file-server` module.

```ts
import { serveDir } from "jsr:@std/http/file-server";

export default {
  fetch: (req: Request) => serveDir(req, {
    fsRoot: "./public",
  }),
}
```

Alternatively, if your static websites is stored on github, you can use the `serveGithubRepo` helper:

```ts
import { serveGithubRepo } from "https://esm.town/v/pomdtr/serveGithubRepo?v=34";

export default {
    fetch: serveGithubRepo({
        owner: "pomdtr",
        repo: "smallweb",
        ref: "bab42e6", // use the commit hash or a branch name (default: main)
    }),
} satisfies Deno.ServeDefaultExport;
```

