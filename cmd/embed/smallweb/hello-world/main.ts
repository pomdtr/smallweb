export default {
    fetch: (req: Request) => {
        const url = new URL(req.url);
        const name = url.searchParams.get("name") || "smallweb";
        return new Response(`Hello, ${name}!`, {
            headers: { "content-type": "text/plain" },
        });
    },
    run: (args: string[]) => {
        const name = args[0] || "smallweb";
        console.log(`Hello, ${name}!`);
    },
};
