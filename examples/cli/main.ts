export default {
    fetch: () => new Response("This app is supposed to be used from the cli: smallweb run cli"),
    run() {
        console.log("Hello, world!");
    }
}
