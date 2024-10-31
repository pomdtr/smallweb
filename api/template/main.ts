export default {
    fetch: () => {
        return new Response("Hello from smallweb!");
    },
    run: () => {
        console.log("Hello from smallweb!");
    },
};
