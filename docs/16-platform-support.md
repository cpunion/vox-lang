# 平台支持

目标平台（当前实现）：

- Linux: `linux-x86`, `linux-amd64`, `linux-arm64`
- macOS: `darwin-amd64`, `darwin-arm64`
- Windows: `windows-x86`, `windows-amd64`, `windows-arm64`
- WASM: `wasm-wasm32`（三元组：`wasm32-unknown-unknown` / `wasm32-wasi` / `wasm32-unknown-emscripten`）

## CLI

`vox build/test/install`（兼容别名：`build-pkg/test-pkg`）支持目标平台参数：

- `--target=<value>`
- `--target <value>`

`vox build/install`（兼容别名：`build-pkg/install-pkg`）额外支持产物参数：

- `--artifact=<kind>`
- `--artifact <kind>`

`<kind>`: `exe`（默认）| `static` | `shared`

`<value>` 可用形式：

- 规范形式：`<os>-<arch>`（例如 `linux-x86`）
- 常见 triple（会自动归一）：
  - `i686-unknown-linux-gnu`
  - `x86_64-unknown-linux-gnu`
  - `aarch64-unknown-linux-gnu`
  - `x86_64-apple-darwin`
  - `aarch64-apple-darwin`
  - `i686-w64-mingw32`
  - `x86_64-w64-mingw32`
  - `aarch64-w64-mingw32`
  - `i686-pc-windows-msvc`
  - `x86_64-pc-windows-msvc`
  - `aarch64-pc-windows-msvc`
  - `wasm32-unknown-unknown`
  - `wasm32-wasi`
  - `wasm32-unknown-emscripten`
- `host` / `native` / 空值：使用当前宿主平台

说明：`emit-c` / `emit-pkg-c` / `list-pkg` / `fmt` / `lsp` 不接受 `--target`。

## Windows ABI

Windows 目标支持两条工具链：

- GNU / MinGW (`w64-mingw32`)
- MSVC (`pc-windows-msvc`)

解析规则：

- `--target` 包含 `msvc` 时归一为 MSVC 路径
- `--target` 包含 `mingw` / `gnu` / `w64` 时归一为 GNU 路径
- 仅给 `windows-<arch>` 时默认 GNU 路径

## C 编译器选择

按以下优先级选择 C 编译器：

1. `CC`
2. `CC_<OS_ARCH>`（例如 `CC_WINDOWS_X86`）
3. `CC_<TRIPLE>`（例如 `CC_I686_W64_MINGW32` / `CC_I686_PC_WINDOWS_MSVC`）
4. 内置默认

内置默认：

- 同宿主目标：
  - Windows GNU 默认 `gcc`
  - Windows MSVC 默认 `cl`
  - 其他平台默认 `cc`
- 交叉目标：
  - `windows-x86` GNU -> `i686-w64-mingw32-gcc`
  - `windows-amd64` GNU -> `x86_64-w64-mingw32-gcc`
  - `windows-arm64` GNU -> `aarch64-w64-mingw32-gcc`
  - `windows-*` MSVC -> Windows 宿主默认 `cl`，非 Windows 宿主默认 `clang --target=<msvc-triple>`
  - `linux-x86` -> `i686-linux-gnu-gcc`
  - `linux-amd64` -> `x86_64-linux-gnu-gcc`
  - `linux-arm64` -> `aarch64-linux-gnu-gcc`
  - `darwin-*` -> `clang --target=<triple>`
  - `wasm-wasm32`:
    - `wasm32-wasi` -> `clang --target=wasm32-wasi`
    - `wasm32-unknown-unknown` / `wasm32-unknown-emscripten` -> `emcc`
    - 以上均可通过 `CC` 覆盖

## 组合约束

- 当前不支持 `darwin-x86`（已在 `--target` 解析阶段拒绝）。
- `test`（兼容别名：`test-pkg`）对 wasm 仅支持 `wasm32-wasi`。
- wasm 测试 runner 通过环境变量 `WASM_RUNNER` 指定（默认 `wasmtime`）。
- `wasm` 目标当前仅支持 `--artifact=exe`。
- `install`（兼容别名：`install-pkg`）当前仅支持宿主平台目标（不支持跨目标安装）。

## 链接与宏

- `exe`:
  - Windows GNU: `-lws2_32 -static -Wl,--stack,8388608`
  - Windows MSVC: `/link ws2_32.lib /STACK:8388608`
- `shared`:
  - Linux/Windows GNU: `-shared`
  - macOS: `-dynamiclib`
  - Windows MSVC: `/LD`
- `static`: 先生成对象文件，再用 `ar/lib` 打包静态库。
- 非 Windows 自动附加：`-D_POSIX_C_SOURCE=200809L -D_DEFAULT_SOURCE`
- wasm 链接：
  - `wasm32-wasi`: 不附加 `--no-entry/--export-all`
  - 其他 wasm 目标：附加 `-Wl,--no-entry -Wl,--export-all`

## WASM / 嵌入

当前支持通过 C 后端编译到 wasm 目标（实验态）：

- `wasm32-wasi`: 可用于 `test`（由 `WASM_RUNNER` 执行）
- `wasm32-unknown-unknown`: 仅构建
- `wasm32-unknown-emscripten`: 仅构建
