import { encodeBase64, decodeBase64 } from "jsr:@std/encoding/base64";

/**
 * Given a Response object, serialize it.
 * Note: if you try this twice on the same Response, it'll
 * crash! Streams, like resp.arrayBuffer(), can only
 * be consumed once.
 */
export async function serializeResponse(resp: Response) {
  return {
    code: resp.status,
    headers: [...resp.headers.entries()],
    body: resp.body ? encodeBase64(await resp.arrayBuffer()) : null,
  };
}

type SerializedRequest = {
  url: string;
  method: string;
  headers: [string, string][];
  body?: string;
};

function deserializeRequest(arg: SerializedRequest) {
  return new Request(arg.url, {
    method: arg.method,
    headers: arg.headers,
    ...(arg.body ? { body: decodeBase64(arg.body) } : {}),
  });
}

function getHandler(mod: { default?: any }):
  | {
      ok: true;
      fetch: (req: Request) => Response | Promise<Response>;
    }
  | { ok: false; error: Error } {
  if (!("default" in mod)) {
    return {
      ok: false,
      error: new Error("Mods require a default export, this mod has none."),
    };
  }

  if (typeof mod.default !== "function") {
    return {
      ok: false,
      error: new Error("The default export must be a function"),
    };
  }

  return { ok: true, fetch: mod.default };
}

const conn = await Deno.connect({
  transport: "tcp",
  port: parseInt(Deno.args[0]),
});

const reader = conn.readable.getReader();
const { value: inputBytes } = await reader.read();
if (!inputBytes) {
  throw new Error("No input bytes");
}
const input = new TextDecoder().decode(inputBytes);

const { entrypoint, req } = JSON.parse(input) as {
  entrypoint: string;
  req: SerializedRequest;
  env: Record<string, string>;
};

/**
 * Send a message to the host.
 */
try {
  const mod = await import(entrypoint);
  const exp = getHandler(mod);
  if (!exp.ok) {
    throw exp.error;
  }
  const resp = await exp.fetch(deserializeRequest(req));
  const writer = conn.writable.getWriter();
  const output = new TextEncoder().encode(
    JSON.stringify({
      type: "response",
      data: await serializeResponse(resp),
    })
  );
  await writer.write(output);
} catch (e) {
  const writer = conn.writable.getWriter();
  const output = new TextEncoder().encode(
    JSON.stringify({
      type: "error",
      data: {
        message: e.message,
        stack: e.stack,
      },
    })
  );
  await writer.write(output);
}
