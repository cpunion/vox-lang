# Vox Language

Vox 是一门 Rust-like 的系统编程语言，当前仓库采用单主线编译器（rolling selfhost）：

- 编译器源码：`src/`
- 可复用包：`pkgs/`
- 示例程序：`examples/`
- 包清单：`vox.toml`
- 锁文件：`vox.lock`
- 构建输出：`target/`

历史引导实现已归档到 `archive` 分支。

## 安装

### 1) 安装 release 二进制（推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/cpunion/vox-lang/main/install.sh | bash
```

安装脚本默认行为：

- 安装到 `~/.vox/bin/vox`
- 安装标准库到 `~/.vox/lib/src/std`
- 默认覆盖安装
- 下载缓存 + 断点续传（默认开启）

常用参数：

```bash
bash install.sh --version v0.2.10
bash install.sh --platform darwin-arm64
bash install.sh --cache-dir ~/.vox/cache/downloads
bash install.sh --no-cache
bash install.sh --local    # 强制本地源码构建安装
bash install.sh --download # 强制 release 二进制安装
```

### 2) 从仓库本地构建安装

```bash
bash install.sh
vox version
```

说明：当 `install.sh` 位于 `vox-lang` 仓库根目录时，默认走本地构建安装（编译器二进制会嵌入源码根路径用于 std/vox 加载）；`curl ... | bash` 默认走 release 下载安装。

## 快速开始

先看版本：

```bash
vox version
```

```bash
cd examples/c_demo
vox build
vox test --run='*'
vox run
vox install
vox run fmt --check
```

## CLI 概览

当前主命令：

```bash
vox build [--driver=user|tool] [--artifact=exe|static|shared] [--target=<value>] [out.bin]
vox test  [--module=<glob>] [--run=<glob>] [--filter=<text>] [--jobs=N|-j N] [--fail-fast] [--list] [--rerun-failed] [--json] [out.bin]
vox run   [--driver=user|tool] [--artifact=exe] [--target=<value>] [--emit-c[=<path>]] [out.bin]
vox run fmt [--check] [path...]
vox install [--target=<value>] [out.bin]
vox list [--json]
vox fmt [--check] [path...]
vox lsp
vox toolchain current|list|install <vX.Y.Z>|use <vX.Y.Z>|pin <vX.Y.Z>
vox version | --version | -V
```

开发态说明（在 `vox-lang` 仓库根目录）：

- `vox fmt`、`vox install` 会先通过 `scripts/ci/rolling-selfhost.sh` 构建本地最新编译器，再由新编译器执行命令。
- `vox run fmt ...` 是 `vox fmt ...` 的别名，便于显式触发“先构建再执行”。

## 常用参数

- `--target=<value>`：`build/test/run/install` 目标平台（如 `darwin-arm64`、`x86_64-w64-mingw32`、`x86_64-pc-windows-msvc`、`wasm32-unknown-unknown`）
- `--artifact=<kind>`：`build/run/install` 产物类型（`exe|static|shared`）
- `--emit-c[=<path>]`：`build/test/run` 输出生成的 C 代码（不带路径时使用默认构建路径）
- `-x`：打印外部命令（如 `cc`/`cp`/测试子进程）后再执行
- `--module=<glob>` / `--run=<glob>` / `--filter=<text>`：测试筛选
- `--jobs=N`：模块级并行测试（模块内串行）
- `--rerun-failed`：只重跑上次失败项
- `--json`：`test/list` JSON 输出

## 环境变量

- `CC` / `CC_<OS_ARCH>` / `CC_<TRIPLE>`：指定 C 编译器
- `VOX_CC_DEBUG=1`：C 编译调试符号
- `VOX_PROFILE=1`：打印编译/链接阶段耗时
- `VOX_ROOT=<path>`：源码根目录覆盖（从 `src/std` 与 `src/vox` 加载）
- `VOX_STDLIB=<path>`：标准库覆盖路径（支持 `<path>`、`<path>/src`、`<path>/src/std`）
- `VOX_NODE=<path>`：`vox lsp` 的 node 可执行路径
- `VOX_TOOLCHAIN`：覆盖当前工具链版本选择
- `VOX_DEV_SELFHOST_DISABLE=1`：关闭仓库开发态 `fmt/install` 本地自举重路由

标准库查找优先级：

1. `VOX_ROOT`（`src/std`）
2. `VOX_STDLIB`
3. `<bin>/../src/std`
4. 构建时嵌入源码目录（`buildinfo.BUILD_SOURCE_ROOT`）
5. 本地开发布局兜底

## 平台支持

见 `docs/internal/16-platform-support.md`。当前目标平台：

- Linux: `linux-x86` / `linux-amd64` / `linux-arm64`
- macOS: `darwin-amd64` / `darwin-arm64`
- Windows: `windows-x86` / `windows-amd64` / `windows-arm64`（GNU/MinGW + MSVC）
- WASM: `wasm32-unknown-unknown` / `wasm32-wasi` / `wasm32-unknown-emscripten`

## 开发与测试

推荐门禁：

```bash
make fmt-check
make test-active
make test-public-api
make test
```

rolling selfhost：

```bash
./scripts/ci/rolling-selfhost.sh build
./scripts/ci/rolling-selfhost.sh test
./scripts/ci/rolling-selfhost.sh print-bin
```

## 工具链管理

```bash
vox toolchain current
vox toolchain list
vox toolchain install v0.2.10
vox toolchain use v0.2.10
vox toolchain pin v0.2.10
```

## LSP 与 VSCode

- LSP 入口：`vox lsp`
- VSCode 扩展源码：`tools/vscode/vox-lang`

## 相关文档

建议阅读顺序：

1. `docs/internal/00-overview.md`
2. `docs/internal/README.md`
3. `docs/internal/15-toolchain.md`
4. `docs/internal/16-platform-support.md`
5. `docs/internal/24-release-process.md`

语言规范与实现细节：

- `docs/internal/01-type-system.md`
- `docs/internal/07-memory-management.md`
- `docs/internal/09-async-model.md`
- `docs/internal/10-macro-system.md`
- `docs/internal/17-ffi-interop.md`
- `docs/internal/19-ir-spec.md`
- `docs/internal/28-vox-libraries.md`
