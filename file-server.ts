import * as fileServer from "jsr:@std/http/file-server";
import esbuild from "esbuild";
import { denoPlugins } from "jsr:@luca/esbuild-deno-loader@^0.10.3";
import * as path from "@std/path";

export async function serveDir(req: Request, params: { fsRoot: string }) {
  const url = new URL(req.url);
  const filepath = path.join(params.fsRoot, url.pathname);

  if ([".tsx", ".ts", ".jsx"].includes(path.extname(filepath))) {
    const result = await esbuild.build({
      jsx: "automatic",
      bundle: false,
      entryPoints: [filepath],
      format: "esm",
      write: false,
      plugins: [
        {
          name: "externals",
          setup: (build) => {
            build.onResolve(
              {
                filter: /.*/,
              },
              (args) => {
                if (args.path !== filepath) {
                  return {
                    path: args.path,
                    external: true,
                  };
                }

                return null;
              }
            );
          },
        },
        ...denoPlugins(),
      ],
    });

    return new Response(result.outputFiles[0].text, {
      headers: {
        "content-type": "text/javascript",
      },
    });
  }

  return fileServer.serveDir(req, params);
}
