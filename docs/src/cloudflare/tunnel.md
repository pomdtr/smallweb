
Cloudflare Tunnel is a free service that allows you to expose your local server to the internet, without having to expose your local IP address.

Additionally, it provides some protection against DDoS attacks, and allows you to use Cloudflare's other services like Access.

## Setup

1. [Install Smallweb](../getting-started.md), and start the evaluation server

    ```bash
    smallweb serve
    ```

1. [Install cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/)

1. Login and create a tunnel.

    ```bash
    cloudflared tunnel login
    cloudflared tunnel create smallweb
    ```

1. Add your domain to cloudflare, and setup a wildcard record pointing to the tunnel. You can find the tunnel id by running `cloudflared tunnel list` command.

    ![alt text](./dns.png)

1. Add the wildcard route in your tunnel config

    ![alt text](./wildcard.png)

## Next Steps

- Protect some hosts with [Access](https://developers.cloudflare.com/cloudflare-one/policies/access/)
