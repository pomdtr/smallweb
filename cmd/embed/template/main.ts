/// <reference types="./smallweb.d.ts" />

export default {
    fetch: () => {
        return new Response("Hello from smallweb!");
    },
    run: () => {
        console.log("Hello from smallweb!");
    },
} satisfies Smallweb.App;
