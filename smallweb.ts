#!/usr/bin/env -S deno run -A

import { createHandler } from "./handler.ts";
import { existsSync } from "@std/fs/exists";
import * as path from "@std/path";
import { serveDir } from "@std/http/file-server";
import * as dotenv from "@std/dotenv";
import { Command } from "cliffy";

new Command()
  .name("smallweb")
  .version("0.0.1")
  .arguments("[rootDir]")
  .option("-p, --port <port:number>", "Port to listen on")
  .action((options, rootDir) => {
    if (!rootDir) {
      rootDir = Deno.cwd();
    }

    const extensions = [".tsx", ".ts", ".jsx", ".js"];
    function inferEntrypoint(root: string, name: string) {
      for (const ext of extensions) {
        const entrypoint = path.join(name + ext);
        if (existsSync(entrypoint)) {
          return entrypoint;
        }
      }

      for (const ext of extensions) {
        const entrypoint = path.join(root, name, "mod" + ext);

        if (existsSync(entrypoint)) {
          return entrypoint;
        }
      }

      const index = path.join(root, name, "index.html");
      if (index) {
        return index;
      }

      return null;
    }

    function loadEnv(root: string, entrypoint: string) {
      if (entrypoint.endsWith(".html")) {
        return {};
      }

      let rootEnv = {};
      const rootEnvPath = path.join(root, ".env");
      if (existsSync(rootEnvPath)) {
        rootEnv = dotenv.loadSync({ envPath: rootEnvPath });
      }

      const envPath = path.join(path.dirname(entrypoint), ".env");
      if (rootEnvPath == envPath) {
        return rootEnv;
      }
      return { ...rootEnv, ...dotenv.loadSync({ envPath }) };
    }

    Deno.serve(
      {
        port: options.port,
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

        const entrypoint = inferEntrypoint(rootDir, val);
        if (!entrypoint) {
          return new Response("Not Found", {
            status: 404,
          });
        }

        const env = loadEnv(rootDir, entrypoint);

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
  })
  .parse(Deno.args);
