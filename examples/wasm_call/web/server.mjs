import http from "node:http";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "..");
const host = "127.0.0.1";
const port = 8080;

const mime = new Map([
  [".html", "text/html; charset=utf-8"],
  [".js", "text/javascript; charset=utf-8"],
  [".wasm", "application/wasm"],
  [".json", "application/json; charset=utf-8"],
  [".css", "text/css; charset=utf-8"],
]);

const server = http.createServer(async (req, res) => {
  try {
    const urlPath = (req.url || "/").split("?")[0];
    let rel = urlPath === "/" ? "/web/index.html" : urlPath;
    if (rel.endsWith("/")) rel = rel.concat("index.html");
    const full = path.resolve(root, "." + rel);

    if (!full.startsWith(root)) {
      res.writeHead(403);
      res.end("forbidden");
      return;
    }

    const data = await readFile(full);
    const ext = path.extname(full);
    res.writeHead(200, { "Content-Type": mime.get(ext) || "application/octet-stream" });
    res.end(data);
  } catch {
    res.writeHead(404);
    res.end("not found");
  }
});

server.listen(port, host, () => {
  console.log(`web server: http://${host}:${port}/web/`);
});
