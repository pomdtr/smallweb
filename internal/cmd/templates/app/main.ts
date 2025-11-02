export default {
    fetch: (_req: Request) => {
        return new Response("Hello, Smallweb!", {
            headers: { "Content-Type": "text/plain" },
        });
    },
    run: (_args: string[]) => {
        console.log("Hello, Smallweb!");
    },
}
