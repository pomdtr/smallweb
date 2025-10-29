import { Smallweb } from "jsr:@smallweb/sdk@0.3.0"

async function handleRequest() {
    const smallweb = new Smallweb("..")

    const entries = await smallweb.apps.list()
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
            ${entries.filter(entry => !entry.name.startsWith(".")).map(app => `
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
    const { SMALLWEB_DIR } = Deno.env.toObject()
    const entries = await Array.fromAsync(Deno.readDir(SMALLWEB_DIR))
    console.log(entries.filter(entry => !entry.name.startsWith(".")).map(entry => entry.name).join("\n"))
}

export default {
    fetch: handleRequest,
    run: handleCommand,
}
