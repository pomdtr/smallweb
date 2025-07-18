export default {
    fetch: async () => {
        const content = await Deno.readTextFile("data/smallweb.log");
        return new Response(content, {
            headers: {
                "Content-Type": "text/jsonl",
                "Cache-Control": "no-cache"
            }
        })
    }
}
