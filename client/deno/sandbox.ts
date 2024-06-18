import { parseArgs } from "jsr:@std/cli";

const { port, entrypoint } = parseArgs(Deno.args, {
  string: ["port", "entrypoint"],
});

if (!port || !entrypoint) {
  console.error(
    "Usage: deno run -A sandbox.ts --port=<port> --entrypoint=<entrypoint>"
  );
  Deno.exit(1);
}

const mod = await import(entrypoint);
if (!mod.default || typeof mod.default !== "object") {
  console.error("Mod does not have a default export");
  Deno.exit(1);
}

const handler = mod.default;
if (!("fetch" in handler) || typeof handler.fetch !== "function") {
  console.error("Mod has no fetch function");
  Deno.exit(1);
}

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
      const resp = await handler.fetch(req);
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
  }
);
