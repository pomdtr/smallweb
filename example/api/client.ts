import { createClient, type NormalizeOAS } from 'npm:fets@0.8.5'
import type openapi from './openapi.ts'

const { SMALLWEB_SOCKET_PATH = "/homo" } = Deno.env.toObject()

const httpClient = Deno.createHttpClient({
    proxy: {
        transport: "unix",
        path: SMALLWEB_SOCKET_PATH,
    }
})

const client = createClient<NormalizeOAS<typeof openapi>>({
    endpoint: '/v1',
})

const response = await client['/blob/{key}'].get({
    params: {
        key: "hello.txt",
    }
})

const pets = await response.json()
console.log(pets)
