# VPS Setup

If you're using a Debian-based VM, you can follow these steps to setup smallweb, assuming you're logged in as root.

These steps will also work on other distributions, but you may need to adjust the package manager commands.

```bash
useradd -m -s $(which bash) smallweb # create user with homedir and default shell
chown smallweb:smallweb /home/smallweb # set the right permissions
passwd smallweb # set a password
usermod -aG sudo smallweb # add sudo access
usermod -aG systemd-journal smallweb # systemd access
loginctl enable-linger smallweb # run service on login
```

At this point, you can switch to the `smallweb` user (ex: using `ssh smallweb@<ip>`) and install smallweb:

```bash
sudo apt update && sudo apt install unzip # install unzip
curl -fsSL https://deno.land/install.sh | sh # install deno
curl -sSfL https://assets.smallweb.run/install.sh | sh # install smallweb
smallweb service install # add smallweb service, and start it
```

To make your service accessible from the internet, you have multiple options:

- setup a reverse proxy on port 443 (ex: caddy)
- using cloudflare tunnel (see [cloudflare setup](./cloudflare/tunnel.md))
