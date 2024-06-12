## Http servers

A smallweb app can be as simple as a single module, exporting a single function. This function will be called with the request object, and should return a response object.

By default, smallweb will look for a module called `http.[js,ts,tsx,jsx]` in the root of the app.

```ts
// basic/http.ts
export default function(req: Request): Response {
    const url = new URL(req.url);
    const name = url.searchParams.get("name") || "world";

    return new Response(`Hello, ${name}!`);
}
```

To run serve this app, you use the smallweb CLI:

```bash
smallweb basic tutorial
```

## Remote Dependencies

To use a dependency from npm or jsr, just import it.

```ts
import { uppercase } from "npm:lodash-es"

export default function(req: Request): Response {
    const url = new URL(req.url);
    const name = url.searchParams.get("name") || "world";

    return new Response(`Hello, ${uppercase(name)}!`);
}

```

## Advanced Routing

Of course, most apps will need more advanced routing. Smallweb integrates well with modern routing libraries like [hono](https://hono.dev).

```ts
import { Hono } from "jsr:@hono/hono";

const app = new Hono();

app.get("/", (c) => {
    return c.json({ message: "Hello, world!" });
});

app.get("/:name", (c) => {
    const { name } = c.param();
    return c.json({ message: `Hello, ${name}!` });
});

export default app.fetch;
```

No need to run a run an install command, deno will automatically fetch the dependencies when you run the app.

Just use `smallweb serve hono` to start the server.

## Using JSX

Smallweb also supports JSX, which allows you to write HTML in your JavaScript files.

```jsx
// jsx/hello.tsx

// this line defines which variant of JSX to use
/** @jsxImportSource https://esm.sh/preact */

import { render } from "https://esm.sh/preact-render-to-string";

export default function(req) {
    const url = new URL(req.url);
    const name = url.searchParams.get("name") || "world";

    return new Response(render(<h1>Hello, {name}!</h1>), {
        headers: { "Content-Type": "text/html" },
    });
}
```

## Serving Static Files

If you only want to serve static files, you can omit the `http` module, and smallweb will serve the files in the root of the app (an `index.html` file need to be present, and will be served by default).

```console
$ tree ~/www/static-example
~/www/static-example
├── index.html
└── style.css
$ cat ~/www/static-example/index.html
<!DOCTYPE html>
<link rel="stylesheet" href="style.css">
<h1>Hello, world!</h1>
```

```bash
smallweb serve static-example
```

## Streaming and Websockets

Smallweb supports streaming responses, and websockets. You can use the `Response` class to create a streaming response, and the `WebSocket` class to create a websocket server.

```ts
export default function(): Response {
  let timer: number | undefined = undefined;
  const body = new ReadableStream({
    start(controller) {
      timer = setInterval(() => {
        const message = `It is ${new Date().toISOString()}\n`;
        controller.enqueue(new TextEncoder().encode(message));
      }, 1000);
    },
    cancel() {
      if (timer !== undefined) {
        clearInterval(timer);
      }
    },
  });
  return new Response(body, {
    headers: {
      "content-type": "text/plain",
      "x-content-type-options": "nosniff",
    },
  });
}
```

## Setting up environment variables

Smallweb supports environment variables, which can be set in the `.env` file in the root of the app.

```console
$ cat ~/www/env-example/.env
PASSWORD=secret
```

## Cli commands

If you create a `cli.[js,ts,tsx,jsx]` module in the root of your app, smallweb will treat it as a CLI command. When you run `smallweb run <app>`, smallweb will call the exported function with the arguments passed on the command line.

```ts
// hello/cli.ts
const [name] = Deno.args;
console.log(`Hello, ${name}!`);
```

```console
$ smallweb run hello Alice
Hello, Alice!
```
