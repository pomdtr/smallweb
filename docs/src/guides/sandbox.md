# App Sandbox

Smallweb apps have access to:

- read and write access to their own directory, and the deno cache directory.
- access to the network, to make HTTP requests.
- access to the env files defined in the global config and in the `.env` file in the app directory.

This sandbox protects the host system from malicious code, and ensures that apps can only access the resources they need.

## Sharing files between apps

To share files between your apps, just use symbolic links!

For example, if you have two apps `app1` and `app2`, and you want `app2` to have access to the files in the `data` directory of `app1`, you can create a symbolic link in the `app2` directory:

```sh
ln -s $HOME/smallweb/app1/data $HOME/smallweb/app2/data
```

Linking files outside the smallweb directory also work, but it causes issues when syncing the files between different machines, so you should only use it if you only want to edit your files on one machine.
