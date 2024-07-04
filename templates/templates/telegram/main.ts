import type { Update } from "npm:@grammyjs/types";

const token = Deno.env.get("BOT_TOKEN");
if (!token) {
    throw new Error("BOT_TOKEN is required");
}

const apiURL = `https://api.telegram.org/bot${token}`;

const handler = async (req: Request) => {
    if (req.method === "GET") {
        return new Response("Telegram bot is up and running");
    }

    const update: Update = await req.json();
    await fetch(`${apiURL}/sendMessage`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            chat_id: update.message?.chat.id,
            text: update.message?.text,
        }),
    });
    return new Response("OK");
};

export default {
    fetch: handler,
};
