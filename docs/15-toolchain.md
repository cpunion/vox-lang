# 工具链

主命令：`vox`（默认构建输出名）

```bash
vox emit-c    [--driver=user|tool] <out.c>   <src...>
vox build     [--driver=user|tool] [--artifact=exe|static|shared] <out.bin> <src...>
vox build-pkg [--driver=user|tool] [--artifact=exe|static|shared] <out.bin>
vox test-pkg  [--module=<glob>] [--run=<glob>] [--filter=<text>] [--jobs=N|-j N] [--fail-fast] [--list] [--rerun-failed] [--json] <out.bin>
vox list-pkg  [--json]
vox toolchain current|list|install <vX.Y.Z>|use <vX.Y.Z>|pin <vX.Y.Z>
vox version | --version | -V
```

`vox version` 解析顺序：`VOX_BUILD_VERSION`（若设置）优先；否则在 git 仓库中推导为 `X.Y.Z[-dirty]-<n>+g<sha>`（命中 tag 且干净时输出 `X.Y.Z`）；无 git 时 `release` 构建输出 `X.Y.Z`，普通源码包输出 `X.Y.Z+src`。

产物类型：`--artifact=exe|static|shared`，默认 `exe`。

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
- `scripts/ci/verify-p0p1.sh`

## 发布与滚动自举

- 唯一链路：`locked release compiler -> new compiler`
- 锁文件：`scripts/release/bootstrap.lock`
- 本地演练：

```bash
make release-dry-run VERSION=v0.2.0-rc1
make release-verify VERSION=v0.2.0-rc1
```

详见：`docs/24-release-process.md`。

## 历史说明

- `stage0` / `stage1` 不再保留在 `main`。
- 历史 bootstrap 代码见分支：`archive/stage0-stage1`。
