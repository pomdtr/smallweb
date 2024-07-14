# Building a Telegram Bot

In this guide, we'll build a simple Telegram bot that will echo back any message that is sent to it.

The telegram API is documented [here](https://core.telegram.org/bots/api).

## Create a new bot

To create a new bot, you will need to talk to the [BotFather](https://t.me/botfather) on Telegram.

1. Open Telegram and search for `@botfather`.
2. Send `/newbot` command to the BotFather.
3. Follow the instructions to create a new bot.
4. Once you have created the bot, the BotFather will give you a token. Save this token as you will need it later

## Create the webhook handler

Create a new file at `~/smallweb/localhost/telegram-echo-bot/main.ts` with the following content:

```ts
const token = Deno.env.get("BOT_TOKEN");
if (!token) {
    throw new Error("BOT_TOKEN is required");
}

const apiURL = `https://api.telegram.org/bot${token}`;

type Update = {
    update_id: number;
    message?: {
        text?: string;
        chat: {
            id: number;
        };
    };
};

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
```

Then create a .env file at `~/smallweb/localhost/telegram-echo-bot/.env` with the following content:

```txt
BOT_TOKEN=<your-token>
```

## Setup the webhook

```sh
export BOT_TOKEN=<your-bot-token>
export WEBHOOK_URL=https://telegram-echo-bot.<your-domain>
curl -X POST https://api.telegram.org/bot$BOT_TOKEN/setWebhook -d "url=$WEBHOOK_URL"
```

You're all set! Now you can send messages to your bot and it will echo them back to you.

## Next steps

If you want to build more complex bots, you can take a look at the [grammY](https://grammy.dev) library. It provides a high-level API to interact with the Telegram API.

```ts
import { Bot, webhookCallback } from "https://deno.land/x/grammy@v1.26.0/mod.ts";

const token = Deno.env.get("BOT_TOKEN");
if (!token) {
    throw new Error("BOT_TOKEN is required");
}

// Create an instance of the `Bot` class and pass your bot token to it.
const bot = new Bot(token); // <-- put your bot token between the ""

// You can now register listeners on your bot object `bot`.
// grammY will call the listeners when users send messages to your bot.

// Handle the /start command.
bot.command("start", (ctx) => ctx.reply("Welcome! Up and running."));

// Handle other messages.
bot.on("message", (ctx) => ctx.reply("Got another message!"));

export default {
    fetch: webhookCallback(bot)
}
```
