import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit"
import { WebglAddon } from "@xterm/addon-webgl"
import { WebLinksAddon } from "@xterm/addon-web-links"
import { AttachAddon } from "@xterm/addon-attach"
import { nanoid } from "nanoid";


async function main() {
  const terminal = new Terminal({
    cursorBlink: true,
    allowProposedApi: true,
    macOptionIsMeta: true,
    macOptionClickForcesSelection: true,
    fontSize: 13,
    fontFamily: "Consolas,Liberation Mono,Menlo,Courier,monospace",
    // theme: window.matchMedia("(prefers-color-scheme: dark)").matches
    //   ? darkTheme
    //   : lightTheme,
  });

  const webLinksAddon = new WebLinksAddon(
    (event, uri) => {
      // check if cmd key is pressed
      if (event.metaKey || event.ctrlKey) {
        window.open(uri, "_blank");
      }

      window.open(uri, "_self");
    },
  );
  terminal.loadAddon(webLinksAddon);

  terminal.loadAddon(new WebglAddon());

  const fitAddon = new FitAddon();
  terminal.loadAddon(fitAddon);

  terminal.open(document.getElementById("terminal")!);
  fitAddon.fit();

  document.title = "Smalleb - Terminal";
  const terminalID = nanoid();

  const protocol = new URL(window.location.href).protocol
  const websocketUrl = new URL(window.location.href.replace(protocol, protocol === "https:" ? "wss:" : "ws:"));

  websocketUrl.searchParams.set("_payload", JSON.stringify({ id: terminalID, cols: terminal.cols, rows: terminal.rows }));

  const ws = new WebSocket(websocketUrl.toString());

  window.onresize = () => {
    fitAddon.fit();
  };

  terminal.onTitleChange((title) => {
    document.title = title;
  });

  terminal.onResize((size) => {
    const { cols, rows } = size;
    const url = new URL(window.location.href);
    url.searchParams.set("cols", cols.toString());
    url.searchParams.set("rows", rows.toString());

    fetch(window.location.href, {
      method: "PATCH",
      body: JSON.stringify({ cols, rows, id: terminalID }),
    });
  });

  const attachAddon = new AttachAddon(ws);
  terminal.loadAddon(attachAddon);

  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", function () {
      // terminal.options.theme = e.matches ? darkTheme : lightTheme;
    });

  ws.onclose = () => {
    attachAddon.dispose()
    terminal.write("\r\nConnection closed, press any key to close the terminal.\r\n");
    terminal.onData(() => {
      window.close();
    });
  }

  window.onbeforeunload = () => {
    ws.onclose = () => { };
    ws.close();
  };


  window.onfocus = () => {
    terminal.focus();
  };

  terminal.focus();
}

main();
