import { Hono } from 'hono'

const app = new Hono()

app.get('/', (c) => c.text('Hono!'))

export default app
