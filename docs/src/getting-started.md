## Installing smallweb

```bash
# Homebrew (macOS / Linux)
brew install pomdtr/tap/smallweb

# From source
go install github.com/pomdtr/smallweb@latest
```

or download the latest release from the [releases page](https://github.com/pomdtr/smallweb/releases).

## Your first smallweb apps

Smallweb maps each folder in `~/www` to a new app. You can find new smallweb apps by searching for the `smallweb-app` topic on github. You can start by cloning one of these repositories in the `~/www` folder.

- [Simple Landing Page](https://github.com/pomdtr/smallweb-landing-example)

    ```sh
    git clone https://github.com/pomdtr/smallweb-landing-example ~/www/landing
    ```

- [Blog using hono](https://github.com/pomdtr/smallweb-blog-example)

    ```sh
    git clone https://github.com/pomdtr/smallweb-blog-example ~/www/blog
    ```

Once you have cloned a few of them, you can publish them using the free tunneling service:

1. First, create an account with the `smallweb auth signup` command.
2. Then, run `smallweb tunnel` to expose your websites to the internet.
3. Open the an app in your browser by visiting `https://<app>.<your-username>.smallweb.run`.

## Hosting your own smallweb server

The tunneling service is not meant for production use. As such, you are encouraged to host your own smallweb server.

If you want your apps to be available on the internet, you'll need to buy a domain name, and point it to your server. You can find more information on how to do this in the [documentation](./cloudflare/tunnel.md).

If you prefer your to keep your apps local to your device, you can learn how to host your apps as `https://<app>.localhost` address in [this guide](./localhost/localhost.md). This option does not requires a domain name (or a server), but your app will only be available on your local device.
