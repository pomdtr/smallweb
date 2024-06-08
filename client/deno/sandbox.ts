import * as path from "jsr:@std/path";

if (!Deno.env.get("SMALLWEB_ENTRYPOINT")) {
  throw new Error("SMALLWEB_ENTRYPOINT is not set");
}

if (!Deno.env.get("SMALLWEB_PORT")) {
  throw new Error("SMALLWEB_PORT is not set");
}

const server = Deno.serve(
  {
    port: parseInt(Deno.env.get("SMALLWEB_PORT")!),
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

    const mod = await import(
      path.join(Deno.cwd(), Deno.env.get("SMALLWEB_ENTRYPOINT")!)
    );
    if (!mod.default || typeof mod.default !== "function") {
      return new Response("Mod has no default export", { status: 500 });
    }
    const handler = mod.default;

    try {
      const resp = await handler(req);
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
