export default {
    fetch() {
        return Response.json(Deno.env.toObject())
    }
}
