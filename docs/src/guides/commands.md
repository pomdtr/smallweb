# Adding cli commands to your app

To add a cli command to your app, you'll need to add a `run` method to your app's default export.

```ts
// File: ~/smallweb/custom-command/main.ts
export default {
    run(args: string[]) {
        console.log("Hello world");
    }
}
```

Use `smallweb run` to execute the command:

```console
$ smallweb run custom-command
Hello world
```

## Using a cli framework

[Cliffy](https://cliffy.io/) is an excellent Deno CLI framework that you can use to build more complex CLI applications.

You can easily wire it to smallweb:

```ts
import { Command } from "jsr:@cliffy/command@1.0.0-rc.5";

export default {
    run(args: string[]) {
        const name = basename(Deno.cwd());
        const command = new Command().name().action(() => {
            console.log(`Hello ${name}`);
        });


        await command.parse(args);
    }
}
```

## Accessing cli commands from your browser

Smallweb automatically serves its own cli commands at `<cli>.<domain>`.

Positional args are mapped to path segments, and flags are mapped to query parameters.

- `smallweb cron ls --json` becomes `https://cli.<domain>/cron/ls?json`

It also allows you to access your app commands and crons:

- `smallweb run custom-command` becomes `https://cli.<domain>/run/custom-command`
- `smallweb cron trigger daily-task` becomes `https://cli.<domain>/cron/trigger/daily-task`

You can specify stdin by sending a POST request with the body as the input.

```sh
# stdin will be "Hello world"
curl -X POST https://cli.<domain>/run/custom-command --data "Hello world"
```
