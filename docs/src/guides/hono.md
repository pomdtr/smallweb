# Creating Services with Hono

Hono is a fantastic library to create APIs in smallweb. It's a great match for smallweb, and the framework I recommend for creating APIs.

Below are some examples to get you started, adapted from the official [Hono documentation](https://hono.dev/docs/)

## Hello world

Create a new `~/www/hono-example/main.ts` file with the following content:

```ts
import { Hono } from 'jsr:@hono/hono'

const app = new Hono()

app.get('/', (c) => c.text('Hello Deno!'))

export default app;
```

Then navigate to `https://hono-example.localhost` to see the result.

## Routing Requests

```ts
import { Hono } from 'jsr:@hono/hono'

const app = new Hono()

// HTTP Methods
app.get('/', (c) => c.text('GET /'))
app.post('/', (c) => c.text('POST /'))
app.put('/', (c) => c.text('PUT /'))
app.delete('/', (c) => c.text('DELETE /'))

// Wildcard
app.get('/wild/*/card', (c) => {
  return c.text('GET /wild/*/card')
})

// Any HTTP methods
app.all('/hello', (c) => c.text('Any Method /hello'))

// Custom HTTP method
app.on('PURGE', '/cache', (c) => c.text('PURGE Method /cache'))

// Multiple Method
app.on(['PUT', 'DELETE'], '/post', (c) =>
  c.text('PUT or DELETE /post')
)

// Multiple Paths
app.on('GET', ['/hello', '/ja/hello', '/en/hello'], (c) =>
  c.text('Hello')
)

export default app;
```

More information can be found in the [Hono documentation](https://hono.dev/docs/api/routing).

## Middlewares

Hono comes with a mature middleware system:

```ts
import { Hono } from "jsr:@hono/hono";
import { logger } from "jsr:@hono/hono/logger";
import { bearerToken } from "jsr:@hono/hono/bearer-token";

// match any method, all routes
app.use(logger())

// specify path
app.use('/posts/*', bearerToken({
    verify(token) {
        return token === Deno.env.get('BEARER_TOKEN')
    }
}))


app.post('/posts', (c) => {
  return c.json({ message: 'Post created' })
})

export default app;
```

More information can be found in the [Hono documentation](https://hono.dev/docs/guides/middleware).

## Serving static files

To serve static files, use `serveStatic` imported from `hono/deno`.

```ts
import { Hono } from 'hono'
import { serveStatic } from 'hono/deno'

const app = new Hono()

app.use('/static/*', serveStatic({ root: './' }))
app.use('/favicon.ico', serveStatic({ path: './favicon.ico' }))
app.get('/', (c) => c.text('You can access: /static/hello.txt'))
app.get('*', serveStatic({ path: './static/fallback.txt' }))

export default app;
```

For the above code, it will work well with the following directory structure.

```txt
./www/hono-example
├── favicon.ico
├── main.ts
└── static
    ├── demo
    │   └── index.html
    ├── fallback.txt
    ├── hello.txt
    └── images
        └── dinotocat.png
```

## JSX Support

If you want to return html from your endpoint, hono comes with built-in support for JSX.

```jsx
// @jsxImportSource jsr:@hono/hono/jsx

import { Hono } from 'jsr:@hono/hono'

const app = new Hono()

app.get('/', (c) => {
    return c.html(
        <html>
            <head>
                <title>Hello Deno!</title>
            </head>
            <body>
                <h1>Hello Deno!</h1>
            </body>
        </html>
    )
})

export default app;
```

More information can be found in the [Hono documentation](https://hono.dev/docs/guides/jsx)

## More

This is just a glimpse of what Hono can do. There are many other features available:

- [RPC](https://hono.dev/docs/guides/rpc)
- [Request Validation](https://hono.dev/docs/guides/validation)
- [Error Handling](https://hono.dev/docs/guides/error-handling)
- and more!

Make sure to visit the [Hono documentation](https://hono.dev/docs) for more information.

## Alternatives

If you're not a fan of Hono, you can also use one the following libraries:

- [Oak](https://deno.land/x/oak) used to be the most popular library for creating APIs in Deno
- [itty-router](https://deno.land/x/itty_router) An ultra-tiny API microrouter
