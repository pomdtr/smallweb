export default {
    fetch: (_req: Request) => {
        return new Response("Hello from smallweb!")
    },
    run: (_args: string[]) => {
        console.log("Hello from smallweb!")
    }
}