This page will guide you through the process of setting up your local environment for smallweb on MacOS.

At the end of this process, each folder in `~/www` will be mapped to domain with a `.localhost` suffix. For example, the folder `~/www/example` will be accessible at `https://example.localhost`.

This setup is useful for developing and testing smallweb apps locally, without having to deploy them to the internet.

If you want to expose your apps to the internet instead, you can follow the [Cloudflare Tunnel setup guide](../cloudflare/tunnel.md).

## Installation

In the future, we might provide a script to automate this process, but for now, it's a manual process.

### Install Brew (required to install smallweb, deno, caddy, and dnsmasq)

```sh
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### Install Deno (required to run smallweb apps)

```sh
brew install deno
```

### Install smallweb

```sh
brew install pomdtr/tap/smallweb

# run smallweb in the background
smallweb service install
```

### Install Caddy (redirect *.localhost to localhost:7777)

```sh
# Install caddy
brew install caddy

# Write caddy configuration
cat <<EOF > /opt/homebrew/etc/Caddyfile
*.localhost {
  tls internal {
    on_demand
  }

  reverse_proxy localhost:7777
}
EOF

# Run caddy in the background
brew services start caddy

# Add caddy https certificates to your keychain
caddy trust
```

### Install dnsmasq (map *.localhost address to 127.0.0.1)

```sh
# Install dsnmasq
brew install dnsmasq

# Write dnsmasq configuration
echo "address=/.localhost/127.0.0.1" > /opt/homebrew/etc/dnsmasq.conf

# Run dnsmasq in the background
sudo brew services start dnsmasq

# Indicates to use dnsmasq for .localhost domains
sudo mkdir -p /etc/resolver
cat <<EOF | sudo tee -a /etc/resolver/localhost
nameserver 127.0.0.1
EOF
```
