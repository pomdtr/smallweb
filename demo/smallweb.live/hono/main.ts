import { Hono } from 'hono'

const app = new Hono()

app.get('/', (c) => c.text('Re!'))

export default app
