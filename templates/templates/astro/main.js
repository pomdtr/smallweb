import { handle } from "./dist/server/entry.mjs";
import { serveDir } from "jsr:@std/http/file-server";

export default {
    async fetch(req) {
        // try to serve static files from the client directory
        const res = await serveDir(req, {
            fsRoot: "./app/dist/client",
        });
        if (res.status !== 404) {
            return res;
        }

        // if the file was not found, pass the request to the app
        return handle(req);
    },
};
