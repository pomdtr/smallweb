import { serveDir } from "jsr:@std/http/file-server";

const handler = (req: Request) => {
    return serveDir(req, {
        fsRoot: "./frontend/dist",
    });
};

export default {
    fetch: handler,
};
