# Integrating with Astro

## Project Structure

Our project will have the following structure:

```txt
~/www/astro-example
├── .vscode/
│   └── settings.json
├── main.ts
└── app/
    └── <astro project>
```

Let's create the project, and initialize the astro frontend:

```sh
mkdir -p ~/www/astro-example
cd ~/www/astro-example
npm create astro@latest app
```

In order to enable ssr, we'll need run `npm install @deno/astro-adapter`, then update the astro.config.mjs file:

```ts
import { defineConfig } from 'astro/config';
import deno from "@deno/astro-adapter";


// https://astro.build/config
export default defineConfig({
    output: "server",
    adapter: deno({
        start: false,
    })
});
```

We'll also create the main.ts file:

```ts
import { handle } from "./app/dist/server/entry.mjs";
import { serveDir } from "jsr:@std/http/file-server";

export default {
    async fetch(req: Request) {
        // try to serve static files from the client directory
        const res = await serveDir(req, {
            fsRoot: "./app/dist/client",
        });
        if (res.status !== 404) {
            return res;
        }

        // if the file was not found, pass the request to the app
        return handle(req);
    },
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

Now that everything is setup, we can just run `npm run build` in the `~/www/astro-example/app` directory, and access the website at `https://astro-example.localhost`.
