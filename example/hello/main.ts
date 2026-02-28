import { Hono } from "npm:hono@4.11.7"

const app = new Hono()

app.post("/hooks/cli", (c) => {
    return c.text(`Hello, ${c.req.query("name") || "world"}!`, 400)
})

export default app