import { ensureDir } from "jsr:@std/fs@^1.0.15/ensure-dir";
import PostalMime from "npm:postal-mime@2.4.3"

export default {
    async email(data: ReadableStream) {
        await ensureDir("./data")

        const msg = await PostalMime.parse(data);
        await Deno.writeTextFile(`./data/email.json`, JSON.stringify(msg, null, 2));
    }
}
