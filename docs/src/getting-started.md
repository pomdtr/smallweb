## Installing smallweb

```bash
# Homebrew (macOS / Linux)
brew install pomdtr/tap/smallweb

# From source
go install github.com/pomdtr/smallweb@latest
```

or download the latest release from the [releases page](https://github.com/pomdtr/smallweb/releases).

## Initializing the `www` folder

```bash
smallweb init
```

This will create a new folder in your home directory called `www`, and a few examples websites in it to get you started.

## Serving the websites

There are a few options to host the websites:

- [Setup a local environment](./localhost/localhost.md)
- [With Cloudflare Tunnel](./cloudflare/tunnel.md)

But the simplest way to get started is to use the tunneling service provided by smallweb:

1. Create a smallweb account by running `smallweb auth login`

2. Start a new tunnel with `smallweb tunnel`

You can access a dashboard listing all your apps at `https://<username>.smallweb.run`, and all your websites will be served from `https://<app>-<username>.smallweb.run`.
