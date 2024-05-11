#!/usr/bin/env -S deno run -A

import { createHandler } from "./handler.ts";
import { existsSync } from "jsr:@std/fs/exists";
import * as path from "jsr:@std/path";
import { serveDir } from "./file-server.ts";
import { config } from "./config.ts";
import * as dotenv from "jsr:@std/dotenv";

const extensions = [".tsx", ".ts", ".jsx", ".js"];
function inferEntrypoint(name: string): {
  entrypoint: string;
  env: Record<string, string>;
} | null {
  for (const ext of extensions) {
    const entrypoint = path.join(config.root, name + ext);
    if (existsSync(entrypoint)) {
      return { entrypoint, env: config.env || {} };
    }
  }

  for (const ext of extensions) {
    const entrypoint = path.join(config.root, name, "mod" + ext);
    if (existsSync(entrypoint)) {
      const envPath = path.join(config.root, name, ".env");
      return {
        entrypoint,
        env: { ...config.env, ...dotenv.loadSync({ envPath }) },
      };
    }
  }

  if (existsSync(path.join(config.root, name, "index.html"))) {
    return { entrypoint: path.join(config.root, name, "index.html"), env: {} };
  }

  return null;
}

Deno.serve(
  {
    port: config.port || 7777,
  },
  async (req) => {
    const url = new URL(req.url);
    const host =
      req.headers.get("x-forwarded-host") ||
      req.headers.get("host") ||
      url.host;
    if (!host) {
      return new Response("No host", {
        status: 400,
      });
    }
    const val = host!.split(".")[0];
    if (!val) {
      return new Response("No val", {
        status: 400,
      });
    }

    const res = inferEntrypoint(val);
    if (!res) {
      return new Response("Not Found", {
        status: 404,
      });
    }
    const { entrypoint, env } = res;

    if (!entrypoint) {
      return new Response("Not Found", {
        status: 404,
      });
    }

    if (path.basename(entrypoint) == "index.html") {
      return serveDir(req, {
        fsRoot: path.dirname(entrypoint),
      });
    }

    const handler = createHandler({
      entrypoint,
      env,
    });
    try {
      const resp = await handler.fetch(req);
      resp.headers.set("access-control-allow-origin", "*");
      return resp;
    } catch (e) {
      return new Response(e.stack, {
        status: 500,
      });
    }
  }
);
