export default {
    fetch: (_req: Request) => {
        return new Response('Hello World!', {
            headers: { 'content-type': 'text/plain' },
        });
    },
    run: (_args: string[]) => {
        console.log('Hello World!');
    }
}
