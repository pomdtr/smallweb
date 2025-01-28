import { accepts } from "jsr:@std/http@1.0.12/negotiation"
import { escape } from "jsr:@std/html@1.0.3"

function cleanStack(str?: string) {
    if (!str) return undefined;
    return str
        .split("\n")
        .filter(
            (line) =>
                !line.includes(import.meta.url) &&
                !line.includes("deno_http/00_serve.ts") &&
                !line.includes("core/01_core.js")
        )
        .join("\n");
}

function serializeError(e: Error) {
    return { name: e.name, message: e.message, stack: cleanStack(e.stack) };
};

function respondWithError(request: Request, error: Error) {
    const e = serializeError(error);
    if (accepts(request, "text/html")) {
        return new Response(/* html */`<!DOCTYPE html>
    <html>
      <head>
        <title>Error</title>
        <style>
          * { box-sizing: border-box }
          body {
            margin: 0;
            font-family: monospace;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            color: black;
            background-color: white;
          }
          div {
            padding: 0 16px;
            width: 100%;
            margin: auto;
            max-width: 768px;
            color: inherit;
            border-left: 0.25em solid #3d444d;
            border-color: #da3633;
          }
          h1 {
            font-weight: 500;
            color: #f85149;
          }
          pre {
            margin: 0;
            padding-bottom: 16px;
            font-size: 12px;
            line-height: 1.5;
            overflow: auto;
            white-space: pre-wrap;
            width: 100%;
          }
        </style>
      </head>
      <body>
        <div>
          <h1>
            ${escape(e.name)}
          </h1>
          <pre>${escape(e.stack ?? e.message)}</pre>
        </div>
      </body>
    </html>`, { status: 500, headers: { 'Content-Type': 'text/html' } });
    }

    return Response.json(
        { error: e },
        { status: 500, headers: { 'Content-Type': 'application/json' } },
    );
}

const input = JSON.parse(Deno.args[0]);

if (!input || !input.command) {
    console.error("Invalid input.");
    Deno.exit(1);
}

if (input.command === "fetch") {
    Deno.serve(
        {
            port: parseInt(input.port),
            onListen: () => {
                // This line will signal that the server is ready to the go
                console.error("READY");
            },
        },
        async (req) => {
            try {
                const mod = await import(input.entrypoint);
                if (!mod.default) {
                    return new Response("The app does not provide a default export.", { status: 500 });
                }

                if (typeof mod.default !== "object") {
                    return new Response("The app default export must be an object.", { status: 500 });
                }
                if (
                    !("fetch" in mod.default) ||
                    typeof mod.default.fetch !== "function"
                ) {
                    return new Response("The app default export does not have a fetch method.", { status: 500 });
                }

                const handler = mod.default.fetch;
                // Websocket requests are stateful and should be handled differently
                if (req.headers.get("upgrade") === "websocket") {
                    const resp = await handler(req);
                    if (!(resp instanceof Response)) {
                        return new Response("Fetch handler must return a Response object.", { status: 500 });
                    }

                    return resp;
                }

                const url = new URL(req.url);
                const proto = req.headers.get("x-forwarded-proto");
                const host = req.headers.get("x-forwarded-host");
                const resp = await handler(new Request(`${proto}://${host}${url.pathname}${url.search}`, {
                    method: req.method,
                    headers: req.headers,
                    body: req.body,
                }));
                if (!(resp instanceof Response)) {
                    throw new Error("Fetch handler must return a Response object.");
                }

                return resp;
            } catch (e) {
                return respondWithError(req, e as Error);
            }
        },
    );
} else if (input.command === "run") {
    const mod = await import(input.entrypoint);
    if (!mod.default || typeof mod.default !== "object") {
        console.error(
            "The mod does not provide an object as it's default export.",
        );
        Deno.exit(1);
    }

    const handler = mod.default;
    if (!("run" in handler)) {
        console.error("The mod default export does not have a run function.");
        Deno.exit(1);
    }

    if (!(typeof handler.run === "function")) {
        console.error("The mod default export run property must be a function.");
        Deno.exit(1);
    }

    await handler.run(input.args);
} else {
    console.error("Unknown command: ", input.command);
    Deno.exit(1);
}
