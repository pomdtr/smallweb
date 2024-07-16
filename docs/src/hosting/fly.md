# Hosting on Fly.io

## Install fly CLI

```sh
# install fly CLI
curl -L https://fly.io/install.sh | sh

# login to fly
fly auth login
```

## Clone the smallweb repository

```sh
git clone https://github.com/pomdtr/smallweb && cd smallweb
```

## Edit the fly.toml file

You'll need to edit the `fly.toml` file to include your app name and region.

```toml
app = "pomdtr-smallweb"
primary_region = "cdg"
```

## Deploy to fly

```sh
fly launch --no-deploy

# add your public key to the fly secrets
fly secrets set "AUTHORIZED_KEYS=$(cat ~/.ssh/id_rsa.pub)"

fly certs create <your-domain>
fly certs create '*.<your-domain>'

# optional - allocate a static V4 IP
fly ips allocate-v4

fly deploy
```

## Access your smallweb instance

```sh
ssh fly@<app-name>.fly.dev
```
