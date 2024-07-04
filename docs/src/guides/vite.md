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
import { Hono } from "jsr:@hono/hono";
import { serveStatic } from "jsr:@hono/hono/deno"

const app = new Hono();

// example api endpoint
app.get("/api/hello", (c) => {
  return c.json({ body: "Hello, World!" });
});

// serve the frontend
app.get('*', serveStatic({ root: './frontend/dist' }))

export default app;
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
