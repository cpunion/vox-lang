# 工具链

主命令：`vox`（默认构建输出名）

```bash
vox build     [--driver=user|tool] [--artifact=exe|static|shared] [--target=<value>] [out.bin]
vox test      [--module=<glob>] [--run=<glob>] [--filter=<text>] [--jobs=N|-j N] [--fail-fast] [--list] [--rerun-failed] [--json] [out.bin]
vox run       [--driver=user|tool] [--artifact=exe] [--target=<value>] [--emit-c[=<path>]] [out.bin]
vox run fmt   [--check] [path...]
vox install   [--target=<value>] [out.bin]
vox list      [--json]
vox fmt       [--check] [path...]
vox lsp
vox toolchain current|list|install <vX.Y.Z>|use <vX.Y.Z>|pin <vX.Y.Z>
vox version | --version | -V
```

默认行为（无 `-pkg` 后缀主命令）：

- `vox build`：构建当前项目（`./src`），默认输出 `target/debug/<package_name>`。
- `vox test`：测试当前项目（`./src` + `./tests`），默认输出 `target/debug/<package_name>`。
- `vox run`：按 `vox build` 构建后运行产物（二进制或 `wasm32-wasi` runner）。
- `vox run fmt`：`vox fmt` 别名（开发态便捷入口）。
- `vox install`：先按 `vox build` 构建，再安装到 `~/.vox/bin/<package_name>`（仅宿主目标）。

在 `vox-lang` 仓库根目录，`vox fmt` 与 `vox install` 会先走 `scripts/ci/rolling-selfhost.sh` 构建本地最新编译器，再由新编译器执行命令。
如需关闭该行为可设置：`VOX_DEV_SELFHOST_DISABLE=1`。

`vox version` 解析顺序：`VOX_BUILD_VERSION`（若设置）优先；否则在 git 仓库中推导为 `X.Y.Z[-dirty]-<n>+g<sha>`（命中 tag 且干净时输出 `X.Y.Z`）；无 git 时 `release` 构建输出 `X.Y.Z`，普通源码包输出 `X.Y.Z+src`。

产物类型：`--artifact=exe|static|shared`，默认 `exe`。

生成 C：`vox build/test/run --emit-c[=<path>]`。

标准库定位：

- 优先级 1：`VOX_STDLIB=<path>`（支持 `<path>`、`<path>/src`、`<path>/src/std`）。
- 优先级 2：编译器二进制旁路：`<bin>/../src/std`。
- 优先级 3：构建时嵌入的源码目录（`buildinfo.BUILD_SOURCE_ROOT`）。
- 兜底：本地开发布局自动推导（如仓库 `target/debug` 场景）。

`vox toolchain install` 会同时安装 `bin/vox` 与 `src/std`，确保优先级 2 可直接生效。

## 仓库布局

- 源码：`src/`
- 清单：`vox.toml`
- 锁文件：`vox.lock`
- 构建产物：`target/`

## 仓库门禁

```bash
make test-active   # rolling selfhost + test smoke + public API contract
make test-public-api
make test          # test-active + examples smoke
```

关键脚本：

- `scripts/ci/rolling-selfhost.sh`
  - 默认按源码内容 + bootstrap 指纹计算缓存键；缓存命中时跳过自举重编译，仅复用 `target/debug/vox_rolling`。
  - 可通过 `VOX_SELFHOST_FORCE_REBUILD=1` 强制重建。
- `scripts/ci/check-std-intrinsics.sh`
  - 校验 `src/std` 中使用的保留 intrinsic（`__*`）是否在 `scripts/release/bootstrap-intrinsics.allow` 中。
  - 防止标准库提前依赖尚未进入锁定 bootstrap 的 intrinsic，导致滚动自举首跳失败。

## 发布与滚动自举

- 唯一链路：`locked release compiler -> new compiler`
- 锁文件：`scripts/release/bootstrap.lock`
- 本地演练：

```bash
make release-dry-run VERSION=v0.2.0-rc1
make release-verify VERSION=v0.2.0-rc1
```

详见：`docs/24-release-process.md`。

### 新增 intrinsic 的两阶段策略

由于 rolling bootstrap 会先用“锁定旧版编译器”编译当前源码，新增保留 intrinsic（`__xxx`）时必须分两步：

1. 先落地编译器侧支持（typecheck builtin + codegen/runtime），但标准库暂不直接调用该 intrinsic。
2. 发布一个新版本并更新 `bootstrap.lock` 后，再在标准库/API 中启用该 intrinsic。

否则旧 bootstrap 会在自举第一跳报 `unknown fn: __xxx`。

## 历史说明

- `stage0` / `stage1` 不再保留在 `main`。
- 历史 bootstrap 代码见分支：`archive/stage0-stage1`。
