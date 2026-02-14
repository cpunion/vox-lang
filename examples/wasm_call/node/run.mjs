import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const wasmPath = path.resolve(here, "..", "target", "vox_wasm_demo.wasm");

function makeImports(module) {
  const wasi = {};
  const env = {};

  for (const imp of WebAssembly.Module.imports(module)) {
    if (imp.module === "wasi_snapshot_preview1") {
      if (imp.name === "proc_exit") {
        wasi[imp.name] = (code) => {
          throw new Error(`proc_exit(${code})`);
        };
      } else {
        wasi[imp.name] = () => 52; // ENOSYS
      }
      continue;
    }

    if (imp.module === "env") {
      env[imp.name] = () => 0;
    }
  }

  return {
    wasi_snapshot_preview1: wasi,
    env,
  };
}

const bytes = await readFile(wasmPath);
const module = new WebAssembly.Module(bytes);
const instance = await WebAssembly.instantiate(module, makeImports(module));

const add = instance.exports.vox_add;
const fib = instance.exports.vox_fib;

if (typeof add !== "function" || typeof fib !== "function") {
  throw new Error("missing exports: vox_add / vox_fib");
}

console.log(`[wasm] vox_add(7, 35) = ${add(7, 35)}`);
console.log(`[wasm] vox_fib(10) = ${fib(10)}`);
