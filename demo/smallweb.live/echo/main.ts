export default {
    fetch: async (req: Request) => {
        return Response.json(
            {
                url: req.url,
                method: req.method,
                body: req.method == "POST" ? await req.text() : undefined,
                headers: Object.fromEntries(req.headers.entries())
            }
        )
    }
}
