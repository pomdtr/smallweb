# Smallweb Plugins

The smallweb CLI can be extennded with plugins. To create a new plugin, just add a binary to your PATH that starts with `smallweb-` and the CLI will automatically detect it.

For example, if you create a new `smallweb-open` file with the following content:

```sh
#!/bin/sh

if [ -z "$1" ]; then
  echo "Usage: smallweb open <app>" >&2
  exit 1
fi

exec open "https://$1.localhost"
```

And make it executable with `chmod +x smallweb-open`, you will be able to run `smallweb open my-app` to open `https://my-app.localhost` in your default browser.
