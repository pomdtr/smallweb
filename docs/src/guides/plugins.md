# Smallweb Plugins

The smallweb CLI can be extennded with plugins. To create a new plugin, just add a binary to your PATH that starts with `smallweb-` and the CLI will automatically detect it.

For example, if you create a new `smallweb-choose` file in your PATH with the following content:

```sh
#!/bin/sh

# check if fzf is installed
if ! command -v fzy 2> /dev/null
then
    echo "fzf could not be found" >&2
    echo "Please install fzf to use this script" >&2
    echo "Docs: https://github.com/junegunn/fzf?tab=readme-ov-file#installation" >&2
    exit 1
fi

smallweb list | cut -f1 | fzf | xargs smallweb open
```

And make it executable with `chmod +x smallweb-choose`, you will be able to run `smallweb choose` and get an interactive list of your apps to choose from, which will then be opened in your default browser.

## Example Plugins

[simpl-site](https://github.com/iamseeley/simpl-site) can be installed as a smallweb plugin. You can install it using the following command:

```sh
deno install -Agf jsr:@iamseeley/simpl-site/smallweb-simpl-site
```

You will then be able to run `smallweb simpl-site` to create a new static site.
