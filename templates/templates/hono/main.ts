import { Hono } from "jsr:@hono/hono";

const app = new Hono();

app.get("/", (c) => {
    return c.text("Hello Hono!");
});

export default app;
