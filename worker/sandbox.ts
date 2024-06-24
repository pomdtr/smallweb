const server = Deno.serve(
  {
    port: parseInt("{{ .Port }}"),
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
      const mod = await import("{{ .ModURL }}");
      if (!mod.default || typeof mod.default !== "object") {
        console.error("Mod does not export a default object");
        Deno.exit(1);
      }

      const handler = mod.default;
      if (!("fetch" in handler) || typeof handler.fetch !== "function") {
        console.error("Mod has no fetch function");
        Deno.exit(1);
      }

      const resp = await handler.fetch(
        new Request("{{ .RequestURL }}", {
          method: req.method,
          headers: req.headers,
          body: req.body,
        }),
      );
      if (!(resp instanceof Response)) {
        return new Response("Mod did not return a Response", { status: 500 });
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
