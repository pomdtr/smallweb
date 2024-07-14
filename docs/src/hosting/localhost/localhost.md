This page will guide you through the process of setting up your local environment for smallweb on MacOS.

This setup is useful for developing and testing smallweb apps locally, without having to deploy them to the internet.

If you want to expose your apps to the internet instead, you can follow the [Cloudflare Tunnel setup guide](../home-server/home-server.md).

## Architecture

The following diagram illustrates the architecture of the local setup:

![Localhost architecture](./architecture.excalidraw.png)

The components needed are:

- a dns server to map `.localhost` domains to `127.0.0.1` ip address (dnsmasq)
- a reverse proxy to automatically generate https certificates for each domain, and redirect traffic to the smallweb evaluation server (caddy)
- a service to map each domain to the corresponding folder in ~/smallweb, and spawn a deno subprocess for each request (smallweb)
- a runtime to evaluate the application code (deno)

## MacOS setup

In the future, we might provide a script to automate this process, but for now, it's a manual process.

### Install Brew

```sh
# install homebrew (if not already installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### Install Deno {#install-deno-macos}

```sh
brew install deno
```

### Setup Smallweb {#setup-smallweb-macos}

```sh
brew install pomdtr/tap/smallweb
smallweb service install
```

### Setup Caddy {#setup-caddy-macos}

```sh
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

mkdir -p ~/smallweb
# Indicate to deno to use the keychain for tls certificates
echo "DENO_TLS_CA_STORE=system" >> ~/smallweb/.env
```

### Setup dnsmasq

```sh
brew install dnsmasq

# Write dnsmasq configuration
echo "address=/.localhost/127.0.0.1" >> /opt/homebrew/etc/dnsmasq.conf

# Run dnsmasq in the background
sudo brew services start dnsmasq

# Indicates to the system to use dnsmasq for .localhost domains
sudo mkdir -p /etc/resolver
cat <<EOF | sudo tee -a /etc/resolver/localhost
nameserver 127.0.0.1
EOF
```

## Testing the setup {#testing-the-setup-macos}

First, let's create a dummy smallweb website:

```sh
mkdir -p ~/smallweb/localhost/example
CAT <<EOF > ~/smallweb/localhost/example/main.ts
export default {
  fetch() {
    return new Response("Smallweb is running", {
      headers: {
        "Content-Type": "text/plain",
      },
    });
  }
}
EOF
```

If everything went well, you should be able to access `https://example.localhost` in your browser, and see the message `Smallweb is running`.

## Ubuntu / Debian setup

### Install Deno {#install-deno-ubuntu}

```sh
curl -fsSL https://deno.land/install.sh | sh
# add ~/.deno/bin to PATH
echo "export PATH=\$PATH:\$HOME/.deno/bin" >> ~/.bashrc
```

### Setup Smallweb {#setup-smallweb-ubuntu}

```sh
curl -FsSL https://install.smallweb.run | sh
# add ~/.local/bin to PATH
echo "export PATH=\$PATH:\$HOME/.local/bin" >> ~/.bashrc
smallweb service install
```

### Setup Caddy {#setup-caddy-ubuntu}

```sh
sudo apt install -y caddy

# Write caddy configuration
cat <<EOF > /etc/caddy/Caddyfile
*.localhost {
  tls internal {
    on_demand
  }

  reverse_proxy localhost:7777
}
EOF

sudo systemctl restart caddy

caddy trust
```

### Testing the setup {#testing-setup-ubuntu}

First, let's create a dummy smallweb website:

```sh
mkdir -p ~/smallweb/localhost/example
CAT <<EOF > ~/smallweb/localhost/example/main.ts
export default {
  fetch() {
    return new Response("Smallweb is running", {
      headers: {
        "Content-Type": "text/plain",
      },
    });
  }
}
EOF
```

If everything went well, you should be able to access `https://example.localhost` in your browser, and see the message `Smallweb is running`.
