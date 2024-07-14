# Deno Deploy

If one of your websites is starting to get a lot of traffic, you might want to deploy it to a cloud provider.

Deno Deploy is a cloud platform that allows you to deploy your Deno apps with ease. It's a great choice for smallweb apps, since it's built on top of Deno, and it's very easy to use.

To deploy an app, you'll just need to:

1. Install the `deployctl` cli:

    ```sh
    deno install -Arf jsr:@deno/deployctl
    ```

2. Run `deployctl deploy`, and follow the instructions.

Beware, all Deno APIs are not available in Deno Deploy. For example, you won't be able to write files to the filesystem.
