async function handleRequest() {
    const { domain } = JSON.parse(await Deno.readTextFile("data/.smallweb/config.json")) as { domain: string }
    const entries = await Array.fromAsync(Deno.readDir("./data"))
    const html = /* html */`
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            border: 1px solid #ddd;
            padding: 8px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
        }
        tr:nth-child(even) {
            background-color: #f2f2f2;
        }
    </style>
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>URL</th>
            </tr>
        </thead>
        <tbody>
            ${entries.filter(entry => !entry.name.startsWith(".")).map(entry => `
                <tr>
                    <td>${entry.name}</td>
                    <td><a href="https://${entry.name}.${domain}">https://${entry.name}.${domain}</a></td>
                </tr>
            `).join('')}
        </tbody>
    </table>
    `
    return new Response(html, {
        headers: {
            "content-type": "text/html"
        }
    })
}

async function handleCommand() {
    const entries = await Array.fromAsync(Deno.readDir("./data"))
    console.log(entries.filter(entry => !entry.name.startsWith(".")).map(entry => entry.name).join("\n"))
}

export default {
    fetch: handleRequest,
    run: handleCommand,
}
