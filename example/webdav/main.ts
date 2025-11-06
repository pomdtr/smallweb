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
        url.host = "webdav.localhost";

        return fetch(
            new Request(url, req),
            { client },
        );
    },
};
