# VPS Setup

If you're using a Debian-based VM, you can follow these steps to setup smallweb, assuming you're logged in as root.

These steps will also work on other distributions, but you may need to adjust the package manager commands.

```bash
# create user with homedir and default shell
useradd -m -s $(which bash) smallweb

# fix home directory permissions
chown smallweb:smallweb /home/smallweb

# set a password for the smallweb user
passwd smallweb

# give the user sudo access
usermod -aG sudo smallweb

# allow the user to use systemd
usermod -aG systemd-journal smallweb

# run user services on login
loginctl enable-linger smallweb
```

At this point, you can switch to the `smallweb` user (ex: using `ssh smallweb@<ip>`) and install smallweb:

```bash
# install unzip (required for deno)
sudo apt update && sudo apt install unzip

# install deno
curl -fsSL https://deno.land/install.sh | sh # install deno

# install smallweb
curl -sSfL https://assets.smallweb.run/install.sh | sh # install smallweb

# start the smallweb service
smallweb service install
```

To make your service accessible from the internet, you have multiple options:

- setup a reverse proxy on port 443 (ex: caddy)
- using cloudflare tunnel (see [cloudflare setup](./cloudflare/tunnel.md))
