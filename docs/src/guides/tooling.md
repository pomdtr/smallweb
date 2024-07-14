# Tooling

Smallweb is not tied to any specific tooling, but here are some tools that I use to develop and deploy my smallweb apps.

## Cloudflare Tunnel

I use [Cloudflare Tunnel](https://www.cloudflare.com/products/tunnel/) to expose my smallweb instances to the internet. It gives me a bunch of features for free:

- DDoS protection
- CDN caching
- Access control (using Cloudflare Access)
- Analytics
- And a bunch more (that I don't know objet)

I evaluated a bunch of tunneling solutions (and even started building my own), but the set of features that Cloudflare provides for free is hard to beat.

## Mutagen

[Mutagen](https://mutagen.io/) is an excellent 2-way file synchronization tool that can keep your local and remote files in sync. It's a great way to develop your smallweb apps locally and deploy them to your server.

Here is how I sync the `smallweb.run` folders between my VPS and my macbook:

```sh
mutagen sync create --name=smallweb-run ~/smallweb.run vps:/home/smallweb/smallweb/smallweb.run
```

## VS Code Remote - SSH

When I want to debug my smallweb apps running on the VPS, I use [VS Code Remote - SSH](https://code.visualstudio.com/docs/remote/ssh). It allows me to connect to my VPS and run the VS Code debugger on the remote machine.

## Tailscale

I use [Tailscale](https://tailscale.com/) to connect all my devices, including my VPS. It allows me to access my smallweb instances from anywhere, as if they were on my local network.

## La Terminal

To edit websites from an ipad or iphone, I recommend using [La Terminal](https://la-terminal.net/). It's the best free option available.

If you're ok with paying a yearly fee for a better experience, I recommend using [Blink Shell](https://blink.sh/).

## Termux

On android, there is no better option than [Termux](https://termux.dev/en/). It's a full-featured terminal emulator that allows me to connect to my VPS and run smallweb apps from my phone.
