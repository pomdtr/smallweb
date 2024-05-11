import { encodeBase64, decodeBase64 } from "@std/encoding/base64";
import { DenoWorker } from "deno-vm";
import * as path from "@std/path";

export type SerializedRequest = {
  url: string;
  method: string;
  headers: [string, string][];
  body?: string | null;
};

export type SerializedResponse = {
  headers: [string, string][];
  status: number;
  statusText: string;
  body: string | null;
};

export async function serializeRequest(
  request: Request
): Promise<SerializedRequest> {
  const body = request.body ? encodeBase64(await request.arrayBuffer()) : null;
  return {
    url: request.url,
    method: request.method,
    headers: Array.from(request.headers.entries()),
    body,
  };
}

export function deserializeResponse(serialized: SerializedResponse): Response {
  const body = serialized.body ? decodeBase64(serialized.body) : null;
  return new Response(body, {
    headers: new Headers(serialized.headers),
    status: serialized.status,
    statusText: serialized.statusText,
  });
}

type InputMsg = {
  entrypoint: string;
  args: unknown[];
  env: Record<string, string>;
};

type OutputMsg = {
  type: "ready" | "return" | "exports" | "error";
  value: any;
};

export function createHandler(params: {
  entrypoint: string;
  env: Record<string, string>;
}) {
  return {
    fetch: async (request: Request) => {
      const serializedRequest = await serializeRequest(request);
      const worker = new DenoWorker(
        new URL("file://" + path.join(import.meta.dirname!, "sandbox.tsx")),
        {
          reload: false,
          spawnOptions: {
            cwd: path.dirname(params.entrypoint),
          },
          permissions: {
            allowAll: true,
          },
        }
      );

      const resp = await new Promise<Response>((resolve, reject) => {
        worker.onmessage = (msg: any) => {
          const data = msg.data as OutputMsg;
          console.log("worker -> host", data.type);
          switch (data.type) {
            case "ready":
              worker.stdout.setEncoding("utf-8");
              worker.stdout.on("data", (data: unknown) => {
                console.log("stdout", data);
              });

              worker.stderr.setEncoding("utf-8");
              worker.stderr.on("data", (data: unknown) => {
                console.error("stderr", data);
              });

              worker.postMessage({
                entrypoint: params.entrypoint,
                env: params.env,
                args: [serializedRequest],
              } satisfies InputMsg);
              break;
            case "return": {
              const response = deserializeResponse(
                data.value as SerializedResponse
              );
              resolve(response);
              break;
            }
            case "error":
              reject(data.value);
              break;
            case "exports":
              console.log("exports", data.value);
              break;
          }
        };
      });

      worker.terminate();
      return resp;
    },
  };
}
