import { cpSync, existsSync, rmSync } from "node:fs";

if (existsSync("dist")) {
    rmSync("dist", { recursive: true });
}

cpSync("node_modules/vscode-web/dist", "dist", { recursive: true });
cpSync("index.html", "dist/index.html");
cpSync("product.json", "dist/product.json");
