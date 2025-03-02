# Smallweb

What if your computer had an internet folder, where each subfolder would be mapped to a subdomains?

Smallweb makes creating a new website as simple as running `mkdir` and creating a few files. Instead of learning with a clunky REST API, you can use the unix tools you already know and love to manage your websites.

## Example

Let's say I want to self-host my own drawing app.

```bash
mkdir -p ~/smallweb/draw
cat <<EOF > ~/smallweb/draw/main.ts
import { Excalidraw } from "jsr:@smallweb/excalidraw@0.9.1";

const excalidraw = new Excalidraw({
    rootDir: "./data"
});

export default excalidraw;
EOF
```

And voila! No need to run a single command, your website is already available at `https://draw.<your-domain>`! And each time the drawing is modified, it get automatically persited to `~/smallweb/draw/data/drawing.json`.

Tired of it ? Just run `rm -rf ~/smallweb/draw` and it's like it never existed.

## Try it out

Leave me a message at [excalidraw.demo.smallweb.live](https://excalidraw.demo.smallweb.live), and then go create your own website from [vscode.demo.smallweb.live](https://vscode.demo.smallweb.live)!

If you want to self-host smallweb, just run `curl https://install.smallweb.run/vps.sh` on a fresh new debian-based server to get a fully working smallweb instance, with on-demand TLS certificates.

Or alternatively, register at [https://cloud.smallweb.run](https://cloud.smallweb.run) to get an hosted instance.

## Application structure

Smallweb tries to keep it's api as simple as possible. The only requirement is to have a default export in a `main.ts` file at the root of your website folder

```typescript
// ~/smallweb/example/main.ts

export default {
    fetch: (request: Request) => {
        return new Response("Example server!");
    },
    run: () => {
        console.log("Example cli!");
    }
}
```

You can invoke:

- the `fetch` function by send a request to `https://example.<your-domain>`
- the `run` function by running `smallweb run example` or `ssh example@<your-domain>`

Of course, it is super easy to hook these functions to a web framweork like [hono](https://hono.dev) or a cli framework like [commander](https://www.npmjs.com/package/commander).

```typescript
// ~/smallweb/hono/main.ts
import { Hono } from 'npm:hono'
const app = new Hono()

app.get('/', (c) => c.text('Hono!'))

export default app
```

And if your app folder does not contain a `main.ts` file, smallweb statically serves the content of the folder instead.

## How it works

Smallweb is distributed as a single golang binary, and use [deno](https://deno.com/) as it's runtime. It uses a serverless architecture, where deno workers are spawned on-demand to handle requests.

If your pet project as no traffic, it won't consume any resources. And if it gets too popular, there is no lock-in, you can easily move to a more traditional hosting provider.

Smallweb leverages deno sandboxing capabilities to isolate each website from each other: then only have access to their own files, and can't interfere with the rest of the system.
