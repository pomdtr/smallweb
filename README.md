<div class="oranda-hide">

# Smallweb - Host websites from your internet folder

</div>

Smallweb is a lightweight web server based on [Deno](https://deno.com). It is inspired both by legacy specifications like [CGI](https://en.wikipedia.org/wiki/Common_Gateway_Interface) and online platforms like [Val Town](https://val.town) and [Deno Deploy](https://deno.com/deploy).

## A self-hosted serveless platform

Smallweb maps each folder in `~/www` to a subdomain: `~/www/example` will be mapped `https://example.localhost` on your local device, and `https://example.<your-domain>` on your homelab / VPS.

Each http request is sandboxed in a single deno subprocess by the smallweb evaluation server. If there is no activity on your website, no resources will be used on your server, making it a great solution for low-traffic websites.

Creating a new website becomes as simple a creating a folder and opening the corresponding url. And since servers are mapped to text files, you can manage them using standard unix tools like `cp`, `mv` or `rm`.

## Installation

All the instructions are written in the [getting started guide](https://pomdtr.github.io/smallweb/book).

Smallweb comes with an (optional) tunneling service, which allows you to quickly expose your websites to the internet without the need to configure your router or buy a domain name. But you are encouraged to use your own domain name and server, as the tunneling service is not meant for production use.

## Demo

The following snippet is stored at `~/www/demo/http.ts` on my raspberrypi 400, and served at <https://demo.pomdtr.me>. Any edit to the file is reflected in real-time, without the need to rebuild the project, or restart the server.

```tsx
/** @jsxImportSource npm:preact */
import { render } from "npm:preact-render-to-string";

export default function () {
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
