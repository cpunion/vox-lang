const log = document.getElementById("log");
const btnAdd = document.getElementById("btn-add");
const btnFib = document.getElementById("btn-fib");

const aInput = document.getElementById("a");
const bInput = document.getElementById("b");
const nInput = document.getElementById("n");

function append(text) {
  log.textContent += `\n${text}`;
}

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

async function load() {
  const resp = await fetch("../target/vox_wasm_demo.wasm");
  if (!resp.ok) {
    throw new Error(`fetch wasm failed: ${resp.status}`);
  }

  const bytes = await resp.arrayBuffer();
  const module = new WebAssembly.Module(bytes);
  const instance = await WebAssembly.instantiate(module, makeImports(module));

  const add = instance.exports.vox_add;
  const fib = instance.exports.vox_fib;

  if (typeof add !== "function" || typeof fib !== "function") {
    throw new Error("missing exports: vox_add / vox_fib");
  }

  btnAdd.disabled = false;
  btnFib.disabled = false;
  log.textContent = "wasm loaded";

  btnAdd.addEventListener("click", () => {
    const a = Number(aInput.value) | 0;
    const b = Number(bInput.value) | 0;
    append(`vox_add(${a}, ${b}) = ${add(a, b)}`);
  });

  btnFib.addEventListener("click", () => {
    const n = Number(nInput.value) | 0;
    append(`vox_fib(${n}) = ${fib(n)}`);
  });
}

load().catch((err) => {
  log.textContent = `load error: ${err.message}`;
  console.error(err);
});
