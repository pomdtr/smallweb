## Installing smallweb

```bash
# Homebrew (macOS / Linux)
brew install pomdtr/tap/smallweb

# From source
go install github.com/pomdtr/smallweb@latest
```

or download the latest release from the [releases page](https://github.com/pomdtr/smallweb/releases).

## Hosting your own smallweb server

If you want your apps to be available on the internet, you'll need to buy a domain name, and point it to your server. You can find more information on how to do this in the [documentation](./cloudflare/tunnel.md).

If you prefer your to keep your apps local to your device, you can learn how to host your apps as `https://<app>.localhost` address in [this guide](./localhost/localhost.md). This option does not requires a domain name (or a server), but your app will only be available on your local device.
