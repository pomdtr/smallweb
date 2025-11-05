import * as path from "jsr:@std/path/posix";
const { SMALLWEB_SOCKET_PATH } = Deno.env.toObject();

const client = Deno.createHttpClient({
  proxy: {
    transport: "unix",
    path: SMALLWEB_SOCKET_PATH,
  },
});

export default {
  fetch: (req: Request) => {
    const url = new URL(req.url);
    return fetch(
      new URL(path.join("/webdav", url.pathname), req.url),
      { client, method: req.method, headers: req.headers, body: req.body },
    );
  },
};
