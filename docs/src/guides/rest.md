# REST API

Each smallweb installation comes with a built-in REST API. You can map it to a subdomain by creating a `smallweb.json` manifest:

```json
// ~/smallweb/api/smallweb.jsonc
{
    "entrypoint": "smallweb:api",
    // make sure to protect your API
    "private": true,
    "publicRoutes": [
        // openapi manifest
        "/openapi.json",
        // json schemas for config files
        "/schemas/*"
    ]
}
```

A swagger UI is available at the root of the api, allowing you to easily test the available endpoints.

## Authentication

You'll need to generate a bearer token to access the API. You can create one by running the following command:

```sh
smallweb token create --description "api token" --app <api-subdomain>
```

You'll then be able to use it to authenticate your requests using this token:

```sh
curl https://<api-domain>/v0/apps -H "Authorization: Bearer <token>"
```

Or you can just use the `smallweb api` command, which automatically authenticates your requests:

```sh
smallweb api /v0/apps
```

## Client Library

Since the API is based on OpenAPI, you can easily generate client libraries for your favorite language.

For usage in smallweb apps, I personally recommend using [feTS](https://the-guild.dev/openapi/fets/client/quick-start).

Here is an example of how you can use it (no code-gen required):

```ts
import { createClient, type NormalizeOAS } from 'npm:fets'
import type openapi from 'jsr:@smallweb/openapi@<smallweb-version>'

const client = createClient<NormalizeOAS<typeof openapi>>({
    endpoint: Deno.env.get("SMALLWEB_API_URL"),
    globalParams: {
        headers: {
            Authorization: `Bearer ${Deno.env.get("SMALLWEB_API_TOKEN")}`
        }
    }
})

const res = await client['/v0/apps'].get()
if (!res.ok) {
    throw new Error(`Failed to fetch apps: ${res.statusText}`)
}

const apps = await res.json() // typed!
```

## Webdav Server

The rest api bundles a webdav server that you can use to manage your files. It is accessible at: `https://<api-domain>/webdav`.

You can easily connect it to any webdav client:

## Windows

1. Open the File Explorer.
2. Click on the `Computer` tab.
3. Click on `Map network drive`.
4. Enter the URL of the webdav server in the `Folder` field.
5. Click `Finish`.
6. Enter your Smallweb username and password.
7. Click `OK`.

## MacOS

1. Open the Finder.
2. Click on `Go` in the menu bar.
3. Click on `Connect to Server`.
4. Enter the URL of the webdav server in the `Server Address` field.
5. Click `Connect`.
6. Enter your Smallweb username and password.
7. Click `Connect`.
8. Click `Done`.

## Linux (Ubuntu)

1. Open Nautilus / Files.
2. Click on `Other Locations`.
3. Enter the URL of the webdav server, and prefix with `davs://` (ex: `davs://<api-domain>/webdav`).
4. Click `Connect`.
5. Enter your Smallweb username and password.
6. Click `Connect`.

## Android

[Material Files](https://play.google.com/store/apps/details?id=me.zhanghai.android.files) has built-in support for WebDAV.

## Javascript

I'm still searching for a good webdav client in javascript. The best I found so far is [webdav-client](https://www.npmjs.com/package/webdav-client), but I don't really like it.

Still, here is an example of how you can use it:

```ts
import * as webdav from 'webdav'

const webdavClient = webdav.createClient(new URL("/webdav", Deno.env.get("SMALLWEB_API_URL")).href, {
    authType: webdav.AuthType.Password,
    username: Deno.env.get("SMALLWEB_API_TOKEN")
})

const apps = await webdavClient.getDirectoryContents("/")
```
