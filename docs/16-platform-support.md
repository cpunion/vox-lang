# 平台支持

目标平台（当前实现）：

- Linux: `linux-x86`, `linux-amd64`, `linux-arm64`
- macOS: `darwin-amd64`, `darwin-arm64`
- Windows: `windows-x86`, `windows-amd64`, `windows-arm64`

## CLI

`vox build/build-pkg/test-pkg` 支持：

- `--target=<value>`
- `--target <value>`

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
- `host` / `native` / 空值：使用当前宿主平台

说明：`emit-c` / `emit-pkg-c` / `list-pkg` 不接受 `--target`。

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

## 组合约束

- 当前不支持 `darwin-x86`（已在 `--target` 解析阶段拒绝）。

## 链接与宏

- Windows GNU 自动附加：`-lws2_32 -static -Wl,--stack,8388608`
- Windows MSVC 自动附加：`/link ws2_32.lib /STACK:8388608`
- 非 Windows 自动附加：`-D_POSIX_C_SOURCE=200809L -D_DEFAULT_SOURCE`

## WASM / 嵌入

WASM 与嵌入制品（`staticlib`/`cdylib`）仍在后续 FFI/目标扩展范围内。
