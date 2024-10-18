# Private Apps

You can automatically protects private apps behind a login prompt. In order to achieve this, you'll need to:

1. Add an `email` field to your global config

    ```json
    // ~/.config/smallweb/config.json
    {
        "domain": "example.com",
        "dir": "~/smallweb",
        "email": "pomdtr@example.com"
    }
    ```

1. Set the private field to true in your app's config.

    ```json
    // ~/smallweb/private-app/smallweb.json
    {
        "private": true
    }
    ```

The next time you'll try to access the app, you'll be prompted with a login screen (provided by lastlogin.net).

Additionaly, you can generate tokens for non-interactive clients using the `smallweb token` create command.

```sh
smallweb token create --description "CI/CD pipeline"
```

Then, you can pass this token in the `Authorization` header of your requests.

```sh
curl https://private-app.example.com -H "Authorization: Bearer <token>"
```

or alternatively, use the basic auth username.

```sh
curl https://private-app.example.com -u "<token>"

# or
curl https://<token>@private-app.example.com
```

If your app is public, but you still want to protect some routes, you can use the `privateRoutes` field in your app's config.

```json
// ~/smallweb/private-app/smallweb.json
{
    "privateRoutes": ["/private/*"]
}
```

There is also a `publicRoutes` field that you can use to protect all routes except the ones listed.

```json
{
    "private": true,
    "publicRoutes": ["/public/*"]
}
```
