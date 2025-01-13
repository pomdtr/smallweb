# Contributing to the project

You'll need to proxy requests from `https://*.smallweb.localhost:443` to `http://localhost:7777` to run the project locally. We'll use caddy to do that.

First, you'll need to install caddy. Then, make sure to add the caddy root certificate to your system's trust store by running `sudo caddy trust`. Finally, use `sudo caddy run` at the root of the project to start the reverse proxy.

If you're using MacOS, you'll also need to create the `/etc/resolver/localhost` file with the following content:

```sh
nameserver 127.0.0.1
```

Then, all debug commands will use the example workspace located in the `examples` folder.
