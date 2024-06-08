<div class="oranda-hide">

# Smallweb - Host websites from your internet folder

</div>

Smallweb simplifies the process of self-hosting websites, taking inspiration from platform like [Deno Deploy](https://deno.com/deploy) or [Val Town](https://val.town).

Creating a website should be as simple as creating a file, or cloning a repository in `~/www`.

## Installation

```sh
# from source
go install github.com/pomdtr/smallweb@latest

# using eget
eget pomdtr/smallweb
```

or download the binary from the [releases page](https://github.com/pomdtr/smallweb/releases).

## Usage

Create a smallweb account using the `smallweb auth signup` command.

The create your first server at `~/www/example/hello.ts`.

```ts
export default function(req: Request) {
    return new Response("Hello, World!");
}
```

Now start a tunnel using the `smallweb tunnel example` command.

Your website will be available at `https://example-<your-username>.smallweb.run`.
