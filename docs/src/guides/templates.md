# Templates

Smallweb comes with a list of templates to help you get started with your project. You can use the `smallweb create` command to create a new project from a template.

```sh
# Interactive mode
smallweb create

# Create a project from a template
smallweb create --name new-app --template hono
```

In addition to this, smallweb is compatible with most static sites generator, so you can use your favorite generator to create your project.

Ex: `npm create vite@latest ~/localhost/my-vite-app`

Depending on the framework you choose, you might need to either:

- Setup your build process to output files in a `dist` folder (smallweb will serve the content of this by default).
- Add a `smallweb.json` file to the root of your project to specify the folder to serve using the `serve` field.
