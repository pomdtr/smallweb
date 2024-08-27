const { entrypoint, args } = JSON.parse(Deno.args[0]);
const mod = await import(entrypoint);
if (!mod.default || typeof mod.default !== "object") {
    console.error("Mod does not export a default object");
    Deno.exit(1);
}

const handler = mod.default;
if (!("run" in handler) || typeof handler.run !== "function") {
    console.error("Mod has no run function");
    Deno.exit(1);
}

await handler.run(args);
