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

type Email = {
  from: string;
  to: string;
  cc: string;
  bcc: string;
  subject: string | undefined;
  text: string | undefined;
  html: string | undefined;
};

function deserializeRequest(arg: SerializedRequest) {
  return new Request(arg.url, {
    method: arg.method,
    headers: arg.headers,
    ...(arg.body ? { body: decodeBase64(arg.body) } : {}),
  });
}

type Input =
  | {
      type: "fetch";
      entrypoint: string;
      req: SerializedRequest;
    }
  | {
      type: "email";
      entrypoint: string;
      email: {};
    };

const conn = await Deno.connect({
  transport: "tcp",
  port: parseInt(Deno.args[0]),
});
const reader = conn.readable.getReader();
const writer = conn.writable.getWriter();
const { value: inputBytes } = await reader.read();
if (!inputBytes) {
  throw new Error("No input bytes");
}
const input = JSON.parse(new TextDecoder().decode(inputBytes)) as Input;

/**
 * Send a message to the host.
 */
try {
  if (input.type === "fetch") {
    const mod = await import(input.entrypoint);
    const handler = mod.default;
    if (!handler || typeof handler !== "function") {
      throw new Error("Mods require a default export, this mod has none.");
    }
    const resp = await handler(deserializeRequest(input.req));
    const output = new TextEncoder().encode(
      JSON.stringify({
        type: "response",
        data: await serializeResponse(resp),
      })
    );
    await writer.write(output);
  } else if (input.type === "email") {
    const mod = await import(input.entrypoint);
    const handler = mod.default;
    if (!handler || typeof handler !== "function") {
      throw new Error("Mods require a default export, this mod has none.");
    }
    await handler(input.email);
    await writer.write(
      new TextEncoder().encode(JSON.stringify({ type: "ok" }))
    );
  }
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
