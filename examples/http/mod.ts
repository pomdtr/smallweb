export default {
  fetch(_req: Request) {
    return new Response("Hello from my macbook", {
      headers: { "content-type": "text/plain" },
    });
  },
};
