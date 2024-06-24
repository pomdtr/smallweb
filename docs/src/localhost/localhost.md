This page will guide you through the process of setting up your local environment for smallweb on MacOS.

At the end of this process, each folder in `~/www` will be mapped to domain with a `.localhost` suffix. For example, the folder `~/www/example` will be accessible at `https://example.localhost`.

## Installation

```bash
# Install Brew (if you haven't already)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Deno
curl -fsSL https://deno.land/install.sh | sh

# Install Smallweb
brew install pomdtr/tap/smallweb

# Install a dummy smallweb app
mkdir ~/www
git clone https://github.com/pomdtr/smallweb-hono-example ~/www/hono-example

# Install caddy
brew install caddy

# *.localhost request will be handled by smallweb
cat <<EOF > /opt/homebrew/etc/Caddyfile
*.localhost {
  tls internal {
    on_demand
  }

  reverse_proxy localhost:7777
}
EOF

# Start the caddy service
brew services start caddy

# Add caddy certificates to your keychain
caddy trust

# Install dsnmasq
brew install dnsmasq

# Redirect *.localhost requests to the local machine
echo "address=/.localhost/127.0.0.1" > /opt/homebrew/etc/dnsmasq.conf

# Start the dnsmasq service
sudo brew services start dnsmasq

# Use dnsmasq to resolve *.localhost requests
sudo mkdir -p /etc/resolver
cat <<EOF | sudo tee -a /etc/resolver/localhost
nameserver 127.0.0.1
EOF

# restart your dns
sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder
```

Now start the smallweb evaluation server:

```bash
smallweb up
```

You should now have access to the hono example at <https://hono-example.localhost>.
