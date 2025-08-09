import { Smallweb } from "https://esm.smallweb.run/sdk@bdf6a3f/pkg/mod.ts"

async function handleRequest() {
    const smallweb = new Smallweb("./data")
    const apps = await smallweb.apps.list()
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
            ${apps.map(app => `
                <tr>
                    <td>${app.name}</td>
                    <td><a href="${app.url}">${app.url}</a></td>
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
