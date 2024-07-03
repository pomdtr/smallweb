# Integrating with Vite

Smallweb can easily integrate with [vite](https://vitejs.dev) to provide a fast development experience.

## Project Structure

Our project will have the following structure:

```txt
~/www/vite-example
├── .vscode/
│   └── settings.json
├── main.ts
└── frontend/
    └── <vite project>
```

Let's create the project, and initialize the vite frontend:

```sh
mkdir -p ~/www/vite-example
cd ~/www/vite-example
npm create vite@latest frontend
cd frontend && npm install && npm run build
```

Then we'll create the main.ts file:

```ts
import { serveDir } from "jsr:@std/http/file-server";

const handler = (req: Request) => {
    return serveDir(req, {
        fsRoot: "./frontend/dist",
    });
};

export default {
    fetch: handler,
};
```

And setup the `.vscode/settings.json` config:

```json
{
    "deno.enable": true,
    "deno.disablePaths": [
        "frontend"
    ],
}
```

Once everything is setup, you should be able to access the website at `https://vite-example.localhost`.

## Adding api endpoints

You can modify the `main.ts` file to add api endpoints. [Hono](https://hono.dev) pairs well with vite for this.

If you need a store small amounts of data, [Deno KV](https://kv.deno.dev) is a good option.

```ts
import { Hono } from "jsr:@hono/hono";

const app = new Hono();

const kv = await Deno.openKv()

// register api endpoints
app.get("/posts", async (c) => {
    const posts = await kv.list(["posts"])
    );

    return c.json(posts.value);
});

app.get("/posts/:id", async (c) => {
    const post = await kv.get(["posts", c.req.param("id")]);
    return c.json(post.value);
});

app.post("/posts", async (c) => {
    const post = await c.req.json();
    await kv.set(["posts", post.id], post);

    return c.json(post, { status: 201 });
});

// serve the frontend
app.get('*', serveStatic({ root: './frontend/dist' }))

export default app;
```
