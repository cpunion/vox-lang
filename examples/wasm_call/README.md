# wasm 调用示例（Node.js + Web）

这个示例演示：

- 用 Vox 导出 wasm 函数（`@ffi_export("wasm", ...)`）
- 在 Node.js 中加载并调用 wasm
- 在浏览器中加载并调用 wasm

## 1. 构建 wasm

在仓库根目录先得到可用编译器（若你还没有 `vox` 命令）：

```bash
./scripts/ci/rolling-selfhost.sh build
```

进入示例目录并构建 wasm：

```bash
cd examples/wasm_call
../../target/debug/vox_rolling build-pkg --target=wasm32-unknown-unknown target/vox_wasm_demo.wasm
```

如果你本地已安装 `vox`，也可直接：

```bash
vox build-pkg --target=wasm32-unknown-unknown target/vox_wasm_demo.wasm
```

## 2. Node.js 调用 wasm

```bash
node node/run.mjs
```

预期输出类似：

```text
[wasm] vox_add(7, 35) = 42
[wasm] vox_fib(10) = 55
```

说明：当前产物会带 `wasi_snapshot_preview1` 导入；示例脚本提供了最小 stub（`ENOSYS`）以便直接调用我们导出的纯计算函数。

## 3. 浏览器调用 wasm

先启动一个静态文件服务（从 `examples/wasm_call` 目录）：

```bash
VOX_WASM_WEB_PORT=8080 node web/server.mjs
```

然后打开（端口默认 8080，可用 `VOX_WASM_WEB_PORT` 覆盖）：

- <http://127.0.0.1:8080/web/>

页面会加载 `target/vox_wasm_demo.wasm`，可在页面里调用 `vox_add` / `vox_fib`。
