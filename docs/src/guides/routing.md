# Routing

Smallweb uses a file-based routing system. This means that the structure of your `smallweb` folder will determine the hostnames your app will respond to.

## Folder structure

Smallweb uses a 2-level folder structure. The first level is the domain, and the second level is the subdomain.

Let's examine the following folder structure:

```txt
~/smallweb/
├── localhost
│   ├── example
│   └── react
├── pomdtr.me
│   └── www
└── smallweb.run
    ├── www
    ├── assets
    └── readme
```

Here 6 websites are defined:

- `example.localhost`
- `react.localhost`
- `www.pomdtr.me`
- `www.smallweb.run`
- `assets.smallweb.run`
- `readme.smallweb.run`

Note that each request for an apex domain (e.g. `pomdtr.me`) will be redirected to the `www` subdomain (e.g. `www.pomdtr.me`).

You can configure the smallweb root folder by setting the `SMALLWEB_ROOT` environment variable.
