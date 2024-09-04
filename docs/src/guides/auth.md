# Private Apps

You can automatically protects private apps behind a login prompt. In order to achieve this, you'll need to:

1. Use the `smallweb token` command to generate a new token.

    ```console
    $ smallweb token
    SF7RZt9shD6UnUcl
    ```

1. Add an `tokens` property to your global config.

    ```json
    // ~/.config/smallweb/config.json
    {
        "tokens": ["SF7RZt9shD6UnUcl"]
    }
    ```

1. Set the private field to true in your app's config.

    ```json
    // ~/smallweb/private-app/smallweb.json
    {
        "private": true
    }
    ```

The next time you'll try to access the app, you'll be prompted with a basic auth dialog.

Use the token as the username, and leave the password empty.

Your tokens are also used to protect internal services that smallweb provides, such as:

- `webdav.<domain>`: A webdav server allowing you to access your files.
- `cli.<domain>`: A web interface to run cli commands.
