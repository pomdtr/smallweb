export default {
    fetch: (_req: Request) => {
        return new Response("Hello from Smallweb!")
    },
    run: (_args: string[]) => {
        console.log("Hello from Smallweb!")
    }
}
