import { encodeBase64, decodeBase64 } from "jsr:@std/encoding/base64";
import { toJson } from "jsr:@std/streams";

/**
 * Given an object of environment variables, create a stub
 * that simulates the same interface as Deno.env
 */
export function createDenoEnvStub(
  input: Record<string, string>
): typeof Deno.env {
  return {
    get(key: string) {
      return input[key];
    },
    has(key: string) {
      return input[key] !== undefined;
    },
    toObject() {
      return { ...input };
    },
    set(_key: string, _value: string) {
      // Stub
    },
    delete(_key: string) {
      // Stub
    },
  };
}

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
      handler: {
        fetch: (req: Request) => Response | Promise<Response>;
      };
    }
  | { ok: false; error: Error } {
  if (!("default" in mod)) {
    return {
      ok: false,
      error: new Error("Mods require a default export, this mod has none."),
    };
  }

  return { ok: true, handler: mod.default };
}

const { entrypoint, env, req, output } = (await toJson(
  Deno.stdin.readable
)) as {
  entrypoint: string;
  req: SerializedRequest;
  env: Record<string, string>;
  output: string;
};

/**
 * Send a message to the host.
 */
try {
  // Override the environment
  Object.defineProperty(Deno, "env", {
    value: createDenoEnvStub(env),
  });

  const mod = await import(entrypoint);
  const exp = getHandler(mod);
  if (!exp.ok) {
    throw exp.error;
  }
  const resp = await exp.handler.fetch(deserializeRequest(req));
  const serialized = await serializeResponse(resp);
  Deno.writeTextFileSync(output, JSON.stringify(serialized));
} catch (e) {
  console.error(e);
  Deno.exit(1);
}
