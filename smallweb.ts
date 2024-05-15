#!/usr/bin/env -S deno run -A

import { createHandler } from "./handler.ts";
import { exists } from "@std/fs/exists";
import * as path from "@std/path";
import { serveDir } from "@std/http/file-server";
import * as dotenv from "@std/dotenv";
import { Command } from "cliffy";

async function loadEnv(root: string, entrypoint: string) {
  if (entrypoint.endsWith(".html")) {
    return {};
  }

  let rootEnv = {};
  const rootEnvPath = path.join(root, ".env");
  if (await exists(rootEnvPath)) {
    rootEnv = dotenv.loadSync({ envPath: rootEnvPath });
  }

  const envPath = path.join(path.dirname(entrypoint), ".env");
  if (rootEnvPath == envPath) {
    return rootEnv;
  }

  const env = await dotenv.load({ envPath });
  return { ...rootEnv, ...env };
}

const extensions = [".tsx", ".ts", ".jsx", ".js"];
async function inferEntrypoint(root: string, name: string) {
  for (const ext of extensions) {
    const entrypoint = path.join(root, name + ext);
    if (await exists(entrypoint)) {
      return entrypoint;
    }
  }

  for (const ext of extensions) {
    const entrypoint = path.join(root, name, "mod" + ext);

    if (await exists(entrypoint)) {
      return entrypoint;
    }
  }

  for (const ext of extensions) {
    const entrypoint = path.join(root, name, name + ext);

    if (await exists(entrypoint)) {
      return entrypoint;
    }
  }

  const index = path.join(root, name, "index.html");
  if (await exists(index)) {
    return index;
  }

  return null;
}

// if the script is running in a remote context,
// we need to fetch the sandbox in order to import local modules from it
async function getSandboxURL() {
  const remoteURL = new URL("sandbox.ts", import.meta.url);
  if (remoteURL.protocol == "file:") {
    return remoteURL;
  }

  const tempDir = await Deno.makeTempDir();
  const localURL = new URL(`file://${tempDir}/sandbox.ts`);
  const resp = await fetch(new URL("sandbox.ts", import.meta.url));
  await Deno.writeTextFile(localURL, await resp.text(), {
    create: true,
  });

  return localURL;
}

new Command()
  .name("smallweb")
  .arguments("[rootDir]")
  .option("-p, --port <port:number>", "Port to listen on")
  .action(async (options, rootDir) => {
    if (!rootDir) {
      rootDir = Deno.cwd();
    }
    const sandboxUrl = await getSandboxURL();
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

        const entrypoint = await inferEntrypoint(rootDir, val);
        if (!entrypoint) {
          return new Response("Could not find entrypoint", {
            status: 404,
          });
        }

        const env = await loadEnv(rootDir, entrypoint);
        if (path.basename(entrypoint) == "index.html") {
          return serveDir(req, {
            fsRoot: path.dirname(entrypoint),
          });
        }

        const handler = createHandler(sandboxUrl, {
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
