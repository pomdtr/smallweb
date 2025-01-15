import PostalMime from 'npm:postal-mime';

export default {
    async email(input: ReadableStream<Uint8Array>) {
        const email = await PostalMime.parse(input);
        await Deno.writeTextFile("data/email.json", JSON.stringify(email, null, 4));
    }
}
