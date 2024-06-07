export default function () {
  return new Response("Hello from my macbook", {
    headers: { "content-type": "text/plain" },
  });
}
