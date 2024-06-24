# Val Town

Val Town is a social website to write and deploy TypeScript.

Val Town and Smallweb can easily interop. For example, to use an http val in smallweb, you can use the following snippet:

```ts
// Assuming that the val export a handler function
import handler from "https://esm.town/v/<username>/<val>"

export default {
  fetch: handler,
};
```

You can also go the other way around. A great way to achieve this is to push you smallweb app to a github repository, and use the raw url to import the app in Val Town.

```ts
import handler from "https://raw.githubusercontent.com/<username>/<repo>/<branch>/mod.ts"

export default handler
```

You can also push reusable block to [JSR](https://jsr.io), and access them from both Val Town and Smallweb.
