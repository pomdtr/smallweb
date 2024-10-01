# VPS / Home Server

If you're using a Debian-based Server, you can follow these steps to setup smallweb, assuming you're logged in as root.

These steps will also work on other distributions, but you may need to adjust the package manager commands.

```bash
# create user with homedir and default shell
useradd --system --user-group --create-home --shell $(which bash) smallweb

# set a password for the smallweb user
passwd smallweb

# give the user sudo access (optional)
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
curl -sSfL https://install.smallweb.run | sh # install smallweb

# start the smallweb service
smallweb service install
```

To make your service accessible from the internet, you have multiple options:

- setup a reverse proxy on port 443 (I use caddy)
- using cloudflare tunnel (see [cloudflare setup](./cloudflare/cloudflare.md))

## Syncing files using mutagen

I recommend using [mutagen](https://mutagen.io) to sync your files between your development machine and the server.

First, install mutagen on your development machine, then enable the daemon using `mutagen daemon register`, and finally, run the following command to sync your files:

```bash
mutagen sync create --name=smallweb --ignore-vcs --ignore=node_modules \
    ~/smallweb smallweb@<server-ip>:/home/smallweb/smallweb
```

From now on, each time you make a change to your files, they will be automatically synced to the server, and vice versa.

Your git repository will only be present on one machine, you can choose if you want to keep it on your development machine or on the server. Syncing git repositories [is not recommended](https://mutagen.io/documentation/synchronization/version-control-systems).

I also prefer to skip syncing the `node_modules` folder, as deno automatically fetches them when needed.
