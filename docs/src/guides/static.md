# Static

## Static Assets

The simplest smallweb app you can create is just a folder with a text file in it.

```sh
mkdir -p ~/smallweb/localhost/example-website
echo "Hello, world!" > ~/smallweb/localhost/example-website/hello.txt
```

If you open `https://hello-world.localhost/hello.txt` in your browser, you should see the content of the file.

## Static Websites

If the folder contains an `index.html` file, it will be served as the root of the website.

```html
<!-- File: ~/smallweb/localhost/example-website/index.html -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Smallweb - Host websites from your internet folder</title>
  <link href="https://cdnjs.cloudflare.com/ajax/libs/tailwindcss/2.2.19/tailwind.min.css" rel="stylesheet">
</head>
<body class="bg-white flex items-center justify-center min-h-screen text-black">
  <div class="border-4 border-black p-10 text-center">
    <h1 class="text-6xl font-extrabold mb-4">Smallweb</h1>
    <p class="text-2xl mb-6">Host websites from your internet folder</p>
  </div>
</body>
</html>
```

If you open `https://example-website.localhost` in your browser, you should see the content of the `index.html` file.

## Static Site Generation

In order to integrate with SSG tools like [Astro](https://astro.build), [Hugo](https://gohugo.io) or [Lume](https://lume.land), smallweb automatically serves the content of the `dist` folder if it contains an `index.html` file.

If you static site generator does not use the `dist` folder, you can either configure it to do so, or create a `smallweb.json` file at the root of your project to specify the folder to serve.

```json
{
  "serve": "public"
}
```
