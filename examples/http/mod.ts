export default {
  fetch(_req: Request) {
    return new Response("Hello world", {
      headers: { "content-type": "text/plain" },
    });
  },
};
