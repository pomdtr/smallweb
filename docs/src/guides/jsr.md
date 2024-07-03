# Installing Apps from JSR

In Smallweb, to install an application, you just import it from [JSR](https://jsr.io). Apps can include both a backend and a frontend, and even a cli.

## SQLite Explorer

Create a new file at `~/www/sqlite-explorer/main.ts` with the following content:

```ts
import { serveDatabase } from "jsr:@pomdtr/sqlite-explorer@0.4.0/server";

const handler = serveDatabase({ dbPath: "./chinook.db" });

export default { fetch: handler };
```

Then download a sample database using:

```txt
curl https://www.sqlitetutorial.net/wp-content/uploads/2018/03/chinook.zip -o /tmp/chinook.zip
unzip /tmp/chinook.zip -d ~/www/sqlite-explorer
```

This application needs some specific permissions to run, so we'll need to configure them in `~/www/sqlite-explorer/smallweb.json`.

```json
{
    "permissions": {
        "read": ["."],
        "write": ["."],
        "ffi": true,
        "sys": true
    }
}
```

Here is what the ~/www/sqlite-explorer folder should look like:

```txt
~/www/sqlite-explorer
├── chinook.db
├── main.ts
└── smallweb.json
```

If you don't want your database to be public, feel free to wrap it in a auth middleware:

```ts
import { serveDatabase } from "jsr:@pomdtr/sqlite-explorer@0.4.0/server";
import { lastlogin } from "jsr:@pomdtr/lastlogin";

const handler = serveDatabase({ dbPath: "./chinook.db" });
const auth = lastlogin({
  verifyEmail: (email) => email === Deno.env.get("EMAIL"),
});

export default { fetch: auth(handler) };
```

And then set the required environment variable in your `.env` file:

```txt
EMAIL=pomdtr@example.com
```

## Visual Studio Code

Create a new file at `~/www/vscode/main.ts` with the following content:

```ts
import { VSCode } from "jsr:@smallweb/vscode@0.0.2";
import { LocalFS } from "jsr:@smallweb/vscode@0.0.2/local-fs";
import { lastlogin } from "jsr:@pomdtr/lastlogin";

const fs = new LocalFS("..");
const vscode = new VSCode({ fs });
const auth = lastlogin({
  verifyEmail: (email) => email === Deno.env.get("EMAIL"),
});

export default {
  fetch: auth(vscode.fetch),
};
```

Here every parts is swappable: you can use a different fs, or a different auth middleware.

For example, the library also provides fs to manage your val.town blobs:

```ts
import { VSCode } from "jsr:@smallweb/vscode@0.0.2";
import { ValTownFS } from "jsr:@smallweb/vscode@0.0.2/val-town";
import { lastlogin } from "jsr:@pomdtr/lastlogin";

const fs = new ValTownFS(Deno.env.get("VALTOWN_TOKEN")!);
const vscode = new VSCode({ fs });
const auth = lastlogin({
  verifyEmail: (email) => email === Deno.env.get("EMAIL"),
});

export default {
  fetch: auth(vscode.fetch),
};
```

To use it, you'll need to set the required environment variables in your `.env` file:

```txt
VALTOWN_TOKEN=<your-token>
```
