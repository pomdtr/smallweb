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
                    console.error(
                        "The app does not provide a default export.",
                    );
                    Deno.exit(1);
                }

                let handler: (req: Request) => Response | Promise<Response>;
                if (typeof mod.default === "object") {
                    if (
                        !("fetch" in mod.default) ||
                        typeof mod.default.fetch !== "function"
                    ) {
                        console.error(
                            "The app default export does not have a fetch function.",
                        );
                        Deno.exit(1);
                    }

                    handler = mod.default.fetch;
                } else if (typeof mod.default === "function") {
                    handler = mod.default;
                } else {
                    console.error(
                        "The app default export must be either an object or a function.",
                    );
                    Deno.exit(1);
                }

                const headers = new Headers(req.headers);
                const url = req.headers.get("x-smallweb-url");
                if (!url) {
                    return new Response("Missing x-smallweb-url header", {
                        status: 400,
                    });
                }
                headers.delete("x-smallweb-url");

                const resp = await handler(
                    new Request(url, {
                        method: req.method,
                        headers,
                        body: req.body,
                    }),
                );
                if (!(resp instanceof Response)) {
                    return new Response(
                        "The app fetch function must return a Response object",
                        {
                            status: 500,
                        },
                    );
                }
                return resp;
            } catch (e) {
                if (e instanceof Error) {
                    console.error(e.stack);
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
