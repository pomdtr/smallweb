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

                if (typeof mod.default !== "object") {
                    console.error(
                        "The app default export must be either an object or a function.",
                    );
                    Deno.exit(1);
                }
                if (
                    !("fetch" in mod.default) ||
                    typeof mod.default.fetch !== "function"
                ) {
                    console.error(
                        "The app default export does not have a fetch method.",
                    );
                    Deno.exit(1);
                }

                const handler = mod.default.fetch;
                return handler(req);
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
