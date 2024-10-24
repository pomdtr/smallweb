export default {
    fetch: () => {
        return new Response('Hello, World!', {
            headers: {
                'content-type': 'text/plain'
            }
        });
    }
}
