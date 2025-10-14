export default {
    async run(_args: string[]) {
        const filePath = "./data/smallweb.log";
        const decoder = new TextDecoder();
        const stat = await Deno.stat(filePath);
        const fileSize = stat.size;

        // Read the last 64KB or less (enough to find 10 lines in most cases)
        const chunkSize = 64 * 1024;
        const readSize = Math.min(chunkSize, fileSize);
        const file = await Deno.open(filePath, { read: true });
        await file.seek(fileSize - readSize, Deno.SeekMode.Start);
        const buf = new Uint8Array(readSize);
        await file.read(buf);
        const text = decoder.decode(buf);

        // Find the start position of the last 10 lines
        const lines = text.split('\n');
        const last10 = lines.slice(-11, -1); // -1 to avoid trailing empty line

        // Print the last 10 lines
        for (const line of last10) {
            await Deno.stdout.write(new TextEncoder().encode(line + "\n"));
        }

        // Now continue tailing the file
        let position = fileSize;
        while (true) {
            const tailBuf = new Uint8Array(1024);
            await file.seek(position, Deno.SeekMode.Start);
            const n = await file.read(tailBuf);
            if (n === null || n === 0) {
                await new Promise((resolve) => setTimeout(resolve, 500));
                continue;
            }
            await Deno.stdout.write(tailBuf.subarray(0, n));
            position += n;
        }
    }
}
