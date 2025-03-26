import { ensureDir } from "jsr:@std/fs@^1.0.15/ensure-dir";
import PostalMime from "npm:postal-mime@2.4.3"

export default {
    async email(msg: Uint8Array) {
        await ensureDir("./data")

        const email = await PostalMime.parse(msg);
        await Deno.writeTextFile(`./data/email.json`, JSON.stringify(email, null, 2));
    }
}
