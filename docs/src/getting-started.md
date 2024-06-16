## Installing smallweb

```bash
# Homebrew (macOS / Linux)
brew install pomdtr/tap/smallweb

# From source
go install github.com/pomdtr/smallweb@latest
```

or download the latest release from the [releases page](https://github.com/pomdtr/smallweb/releases).

## Your first smallweb apps

Smallweb maps each folder in `~/www` to a new app. Feel free to clone one of the following repositories to get started:

TODO: Add a list of smallweb apps

You can find new smallweb apps by searching for the `smallweb-app` topic on github.

Once you have cloned a few of them, you can publish them using the free tunneling service.

First, create an account with the `smallweb auth signup` command. Then, run `smallweb tunnel` to expose your websites to the internet. Each of your apps will be available at `https://<app-name>-<user-name>.smallweb.run`: Every request will be routed to your local device, and handled by the smallweb evaluation server.

## Hosting your own smallweb server

The tunneling service is not meant for production use. As such, you are encouraged to host your own smallweb server.

If you want your apps to be available on the internet, you'll need to buy a domain name, and point it to your server. You can find more information on how to do this in the [documentation](./cloudflare/tunnel.md).

If you prefer your to keep your apps local to your device, you can learn how to host your apps as `https://<app>.localhost` address in [this guide](./localhost/localhost.md). This option does not requires a domain name (or a server), but your app will only be available on your local device.
