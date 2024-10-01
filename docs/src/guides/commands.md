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

const name = basename(Deno.cwd());
const command = new Command().name().action(() => {
    console.log(`Hello ${name}`);
});

export default {
    run(args: string[]) {
        await command.parse(args);
    }
}
```
