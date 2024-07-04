const handler = (_req: Request) => {
    return new Response("Hello, World!", {
        headers: { "content-type": "text/plain" },
    });
};

export default {
    fetch: handler,
};
