# Integrating with Vite

Smallweb can easily integrate with [vite](https://vitejs.dev) to provide a fast development experience.

You can just create a vite project in the `~/smallweb` directory.

```sh
cd ~/smallweb
npm create vite@latest vite-project.localhost
cd vite-project
npm install
npm run build
```

The build command will generate a `dist` directory with the static files. Since your project is in the `~/smallweb/localhost/vite-project` directory, Smallweb will serve the files at `https://vite-project.localhost`.
