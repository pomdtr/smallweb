import PostalMime from 'npm:postal-mime';
import { ensureDir } from "jsr:@std/fs"

export default {
    async email(input: ReadableStream<Uint8Array>) {
        await ensureDir("data");
        const email = await PostalMime.parse(input);
        await Deno.writeTextFile("data/email.json", JSON.stringify(email, null, 4));
    }
}
