# 平台支持

目标平台（当前实现）：

- Linux: `linux-x86`, `linux-amd64`, `linux-arm64`
- macOS: `darwin-amd64`, `darwin-arm64`
- Windows: `windows-x86`, `windows-amd64`, `windows-arm64`
- WASM: `wasm-wasm32`（三元组：`wasm32-unknown-unknown` / `wasm32-wasi` / `wasm32-unknown-emscripten`）

## CLI

`vox build/test/run/install` 支持目标平台参数：

- `--target=<value>`
- `--target <value>`

`vox build/run/install` 额外支持产物参数：

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

说明：`list` / `fmt` / `lsp` 不接受 `--target`；`build/test/run` 通过 `--emit-c` 输出 C 代码。

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
- `test` 对 wasm 仅支持 `wasm32-wasi`。
- wasm 测试 runner 通过环境变量 `WASM_RUNNER` 指定（默认 `wasmtime`）。
- `wasm` 目标当前仅支持 `--artifact=exe`。
- `install` 当前仅支持宿主平台目标（不支持跨目标安装）。

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

## Async Wake Runtime 平台约束（A14-3）

`std/async` 的 `EventRuntime` 基于 `__wake_wait/__wake_notify`。C runtime 当前约束如下：

- Linux: `eventfd + epoll` 事件等待；`notify` 写入 `eventfd`，`wait` 使用 `epoll_wait`。
- macOS/*BSD: `kqueue(EVFILT_USER)` 事件等待；`notify` 使用 `NOTE_TRIGGER`。
- Windows: `IOCP` 事件等待；`notify` 使用 `PostQueuedCompletionStatus`，`wait` 使用 `GetQueuedCompletionStatus`。
- Emscripten: 使用 `sched_yield()`（协作式让出）。
- 其他 POSIX: 使用 `nanosleep` 回退等待。
- 通用语义：
  - `timeout_ms < 0` 会被钳制到 `0`。
  - `token` 未命中 slot 时立即返回 `false`。
  - 返回 `true` 表示本次调用消费到至少一个 pending wake token。

这些分支由 `src/vox/codegen/c_emit_test.vox` 的 wake regression 用例锁定。

## Socket Wait Intrinsics 平台约束（A32-1）

`__tcp_wait_read(handle, timeout_ms)` / `__tcp_wait_write(handle, timeout_ms)` 在 C runtime 的平台分支：

- Linux: `epoll` 单 fd 一次等待（`EPOLLIN/EPOLLOUT` + `ERR/HUP`）。
- macOS/*BSD: `kqueue` 单 fd 一次等待（`EVFILT_READ/EVFILT_WRITE` + `EV_ONESHOT`）。
- Windows: `select` 基线等待（后续升级为 IOCP 语义接线）。
- 其他 POSIX: `select` 回退。

通用语义：

- `timeout_ms < 0` 会被钳制到 `0`。
- 非法句柄会触发 panic。
- 返回 `true` 表示在超时前观察到对应方向的就绪事件。
