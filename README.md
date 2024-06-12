<div class="oranda-hide">

# Smallweb - Host websites from your internet folder

</div>

Smallweb is a lightweight web server based on [Deno](https://deno.com). It is inspired both by [CGI](https://en.wikipedia.org/wiki/Common_Gateway_Interface) and modern serverless platfrom like [Val Town](https://val.town) and [Deno Deploy](https://deno.com/deploy).

Smallweb maps each folder in `~/www` to a subdomain (`~/www/example` will be mapped `https://example.localhost` on your local device, and `https://example.<your-domain>` on your homelab / VPS).

Creating a new website becomes as simple as creating a new folder, or cloning a git repository. Servers are managed using the standard unix tools (ls, mv, rm...). After the initial setup, you never need to worry about build commands, dev servers and ports.

The following snippet is stored at `~/www/demo/http.ts` on my raspberrypi 400, and served at <https://demo.pomdtr.me>. Every update to the file is instantly deployed.

```tsx
/** @jsxImportSource npm:preact */
import { render } from "npm:preact-render-to-string";

export default function (_req: Request) {
  return new Response(
    render(
      <html lang="en">
        <head>
          <meta charset="UTF-8" />
          <meta
            name="viewport"
            content="width=device-width, initial-scale=1.0"
          />
          <title>Smallweb - Host websites from your internet folder</title>
          <link
            href="https://cdnjs.cloudflare.com/ajax/libs/tailwindcss/2.2.19/tailwind.min.css"
            rel="stylesheet"
          />
        </head>
        <body class="bg-white flex items-center justify-center min-h-screen text-black">
          <div class="border-4 border-black p-10 text-center">
            <h1 class="text-6xl font-extrabold mb-4">Smallweb</h1>
            <p class="text-2xl mb-6">Host websites from your internet folder</p>
            <a
              href="https://github.com/pomdtr/smallweb"
              class="px-8 py-3 bg-black text-white font-bold border-4 border-black hover:bg-white hover:text-black transition duration-300"
            >
              Get Started
            </a>
          </div>
        </body>
      </html>,
    ),
    {
      headers: {
        "Content-Type": "text/html",
      },
    },
  );
}
```

You can test smallweb in a few minutes by following the [Getting Started](./getting-started.md) guide.
