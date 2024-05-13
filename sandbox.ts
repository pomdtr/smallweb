import { encodeBase64, decodeBase64 } from "jsr:@std/encoding/base64";

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
    headers: [...resp.headers.entries()],
    body: resp.body ? encodeBase64(await resp.arrayBuffer()) : null,
    status: resp.status,
    statusText: resp.statusText,
  };
}

interface Timing {
  executionStart: number;
  importComplete: number;
  executionComplete: number;
}

interface DoneMessage {
  type: "done";
  wallTime: number;
}
interface ErrorMessage {
  type: "error";
  value: unknown;
}
interface ReadyMessage {
  type: "ready";
}
interface ReturnMessage {
  type: "return";
  value: unknown;
  timing?: Timing;
}
type Message = (DoneMessage | ErrorMessage | ReadyMessage | ReturnMessage) & {
  _send_time?: number;
};

export function send(message: Message) {
  // We used to have a much smaller limit here.
  // This one is just to ensure folks aren't sending crazy amount of data
  try {
    if (JSON.stringify(message).length > 10_000_000) {
      message = {
        type: "error",
        value: {
          name: "WS_PAYLOAD_TOO_LARGE: " + message.type,
          message: `The '${message.type}' event is too large to process`,
        },
      };
    }
  } catch (_) {
    message = {
      type: "error",
      value: {
        name: "WS_PAYLOAD_UNSERIALIZABLE: " + message.type,
        message: `The '${message.type}' event can't be serialized`,
      },
    };
  }
  message._send_time = Date.now();
  (self as any).postMessage(message);
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

const WALL_TIME_START = Date.now();

function getHandler(mod: {
  default?: any;
}):
  | { ok: true; value: (req: Request) => Response | Promise<Response> }
  | { ok: false; error: Error } {
  if (!("default" in mod)) {
    return {
      ok: false,
      error: new Error("Mods require a default export, this mod has none."),
    };
  }

  return { ok: true, value: mod.default };
}

/**
 * Send a message to the host.
 */
self.addEventListener("message", async (msg: any) => {
  if (msg.data.entrypoint) {
    try {
      // Override the environment
      Object.defineProperty(Deno, "env", {
        value: createDenoEnvStub({ ...msg.data.env }),
      });

      const executionStart = performance.now();

      const mod = await import(msg.data.entrypoint);

      const importComplete = performance.now();

      const sendReturn = (value: any) => {
        send({
          type: "return",
          value,
          timing: {
            executionStart,
            importComplete,
            executionComplete: performance.now(),
          },
        });
      };

      const exp = getHandler(mod);
      if (!exp.ok) {
        throw exp.error;
      }
      const resp = await exp.value(deserializeRequest(msg.data.req));
      sendReturn(await serializeResponse(resp));

      // Communicate the result to the parent.
    } catch (e) {
      console.error(e);
      send({ type: "error", value: e });
    }
    send({
      type: "done",
      wallTime: Date.now() - WALL_TIME_START,
    });
  }
});

send({ type: "ready" });
