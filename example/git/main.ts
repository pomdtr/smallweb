const { SMALLWEB_SOCK } = Deno.env.toObject();

const client = Deno.createHttpClient({
    proxy: {
        transport: "unix",
        path: SMALLWEB_SOCK,
    },
});

export default {
    fetch: (req: Request) => {
        const url = new URL(req.url);
        url.pathname = `/git/${url.pathname}`
        return fetch(url, { client, redirect: "manual" });
    },
};
