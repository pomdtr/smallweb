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
        url.host = "api.localhost";

        return fetch(
            new Request(url, req),
            { client, redirect: "manual" },
        );
    },
    run: async () => {
        const resp = await fetch("http://api.localhost/openapi.json", { client })
        console.log(JSON.stringify(await resp.json(), null, 2));
    }
};
