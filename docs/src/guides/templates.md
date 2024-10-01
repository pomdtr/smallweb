# Templates

Smallweb comes with a list of templates to help you get started with your project. You can use the `smallweb init` command to create a new project.

```sh
smallweb app create <name>
```

You can also specify custom a template to use:

```sh
smallweb app create <name> --template pomdtr/smallweb-template-http
```

Any github repository can be used as a template. View a list of the available templates [here](https://github.com/topic/smallweb-template).

To create your own template, just add the `smallweb-template` topic to your repository, and it will be automatically added to the list of available templates.
