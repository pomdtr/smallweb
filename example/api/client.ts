import { createClient, type NormalizeOAS } from 'npm:fets@0.8.5'
import type openapi from './openapi.ts'

const { SMALLWEB_SOCKET_PATH } = Deno.env.toObject()

const httpClient = Deno.createHttpClient({
    proxy: {
        transport: "unix",
        path: SMALLWEB_SOCKET_PATH,
    }
})

export default createClient<NormalizeOAS<typeof openapi>>({
    endpoint: `http://localhost`,
    fetchFn: (input, init) => fetch(input, { ...init, client: httpClient }),
})

