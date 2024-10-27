const input = JSON.parse(Deno.args[0]);

if (input.command === "fetch") {
    const { entrypoint, port } = input;
    const server = Deno.serve(
        {
            port: parseInt(port),
            onListen: () => {
                // This line will signal that the server is ready to the go
                console.log("READY");
            },
        },
        async (req) => {
            // exit the server once the request will be handled
            queueMicrotask(async () => {
                await server.shutdown();
            });
            try {
                const mod = await import(entrypoint);
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
                    if (resp instanceof Response) {
                        return resp;
                    }

                    return new Response("Fetch handler must return a Response object.", { status: 500 });
                }

                const url = new URL(req.url);
                const proto = req.headers.get("x-forwarded-proto");
                const host = req.headers.get("x-forwarded-host");
                const resp = await handler(new Request(`${proto}://${host}${url.pathname}${url.search}`, {
                    method: req.method,
                    headers: req.headers,
                    body: req.body,
                }));
                if (resp instanceof Response) {
                    return resp;
                }

                return new Response("Fetch handler must return a Response object.", { status: 500 });
            } catch (e) {
                if (e instanceof Error) {
                    return new Response(e.stack, { status: 500 });
                }

                return new Response("Unknown error", { status: 500 });
            }
        },
    );
} else if (input.command === "run") {
    const { entrypoint, args } = input;
    const mod = await import(entrypoint);
    if (!mod.default || typeof mod.default !== "object") {
        console.error(
            "The mod does not provide an object as it's default export.",
        );
        Deno.exit(1);
    }

    const handler = mod.default;
    if (!("run" in handler) || typeof handler.run !== "function") {
        console.error("The mod default export does not have a run function.");
        Deno.exit(1);
    }

    await handler.run(args);
} else {
    console.error("Unknown command");
    Deno.exit(1);
}
