const { SMALLWEB_SOCK } = Deno.env.toObject();

const client = Deno.createHttpClient({
    proxy: {
        transport: "unix",
        path: SMALLWEB_SOCK,
    },
});

export default {
    fetch: (req: Request) => {
        return fetch(new Request(req), {
            client, redirect: "manual"
        });
    },
    run: async () => {
        const resp = await fetch("http://api.localhost/v1/apps", { client });
        console.log(JSON.stringify(await resp.json(), null, 2));
    }
};
