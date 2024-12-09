import { Hono } from 'npm:hono@4.6.13'

const app = new Hono()

app.get('/', (c) => c.text('Hono!'))

export default app
