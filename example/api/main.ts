import { Hono } from "npm:hono@4.10.4"
import { Scalar } from 'npm:@scalar/hono-api-reference@0.9.23'

const { SMALLWEB_SOCKET_PATH } = Deno.env.toObject()

const client = Deno.createHttpClient({
    proxy: {
        transport: "unix",
        path: SMALLWEB_SOCKET_PATH,
    }
})

const app = new Hono()

app.get(
    '/docs',
    Scalar({
        url: '/openapi.json',
        theme: 'purple',
    })
)

app.all("*", (c) => {
    const url = new URL(c.req.url)

    return fetch(new URL(url.pathname, "http://api.localhost"), {
        method: c.req.method,
        body: c.req.raw.body,
        headers: c.req.raw.headers,
        client
    })
})

export default {
    fetch: app.fetch
}
