export default {
    fetch: (_req: Request) => {
        return new Response("Welcome to Smallweb!");
    },
    run: (_args: string[]) => {
        console.log("Welcome to Smallweb!");
    },
};
