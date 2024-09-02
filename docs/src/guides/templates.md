# Templates

Smallweb comes with a list of templates to help you get started with your project. You can use the `smallweb init` command to create a new project.

```sh
# create a new project in the current directory
smallweb init
# create a new project in a specific directory
smallweb init <directory>
```

You can also specify custom a template to use:

```sh
smallweb init --template pomdtr/smallweb-template-http
```

Any github repository can be used as a template. View a list of the available templates [here](https://github.com/topic/smallweb-template).

To create your own template, just add the `smallweb-template` topic to your repository, and it will be automatically added to the list of available templates.
