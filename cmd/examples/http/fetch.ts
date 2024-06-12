export default function () {
  return new Response("Hi from smallweb!", {
    headers: { "content-type": "text/plain" },
  });
}
