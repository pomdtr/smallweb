# Custom Domains

To configure a custom domain for your smallweb app, you'll just need to create a `CNAME` file in the root of your project, with the domain name you want to use.

For example, if you want to use `example.com`, create a file named `CNAME` with the following content:

```txt
example.com
```

Then, you'll need to configure your DNS provider to point your domain to the smallweb server. You can do this by creating a `CNAME` record that points to the smallweb server's domain.
