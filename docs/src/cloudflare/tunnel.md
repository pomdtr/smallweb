Cloudflare Tunnel is a **free** service that allows you to expose your local server to the internet, without having to expose your local IP address.

Additionally, it provides some protection against DDoS attacks, and allows you to use Cloudflare's other services like Access.

## Setup

1. Make sure that you have a domain name that you can manage with Cloudflare.

1. Install smallweb on your server, and register it as a service.

    ```ts
    git clone https://github.com/pomdtr/smallweb
    cd smallweb && go install
    smallweb service install
    ```

1. From your cloudflare dashboard, navigate to `Zero Trust > Networks > Tunnels`

1. Click on `Create a tunnel`, and select the `Clouflared` option

1. Follow the intructions to install cloudflared, and create a connector on your device.

1. Add a wildcard hostname for your tunnel (ex: `*.<your-domain>`), and use `http://localhost:7777` as the origin service.

    ![Tunnel Configuration](./tunnel.png)

1. Copy the tunnel ID, and go to `Websites > DNS > Records`.

1. Add a new `CNAME` record for your wildcard hostname, with a target of `<tunnel-id>.cfargotunnel.com`.

    ![DNS Configuration](./dns.png)

## Checking that your tunnel is running

Create a dummy smallweb app in `~/www`

```sh
mkdir -p ~/www/example
CAT <<EOF > ~/www/example/main.ts
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

## Optional Steps

- You can protect your tunnel (or specific apps) with Cloudflare Access.
