# Contributing to the project

You'll need to proxy requests from `*.smallweb.localhost:443` to `localhost:7777` to run the project locally. The easiest to achieve is to use the following caddyfile:

```sh
smallweb.localhost, *.smallweb.localhost {
  reverse_proxy localhost:7777
}
```

You can follow the instructions from the [docs](https://www.smallweb.run/docs/hosting/local/) to find out how to install caddy locally.

Then, all debug commands will use the example workspace located in the `examples` folder.
