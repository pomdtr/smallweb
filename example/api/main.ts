import smallweb from './client.ts'

const { SMALLWEB_SOCKET_PATH } = Deno.env.toObject()

const client = Deno.createHttpClient({
    proxy: {
        transport: "unix",
        path: SMALLWEB_SOCKET_PATH,
    }
})



export default {
    fetch: (req: Request) => {
        return fetch(req, { client })
    },
    async run() {
        const resp = await smallweb["/v1/apps"].get({})
        if (!resp.ok) {
            console.error('Error fetching apps:', resp.statusText)
            return
        }
        const body = await resp.json()
        console.log(body)
    }
}
