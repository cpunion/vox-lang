# 工具链（stage2-only）

主命令：`vox_stage2`（默认构建输出名）

```bash
vox_stage2 emit-c    [--driver=user|tool] <out.c>   <src...>
vox_stage2 build     [--driver=user|tool] <out.bin> <src...>
vox_stage2 build-pkg [--driver=user|tool] <out.bin>
vox_stage2 test-pkg  [--module=<glob>] [--run=<glob>] [--filter=<text>] [--jobs=N|-j N] [--fail-fast] [--list] [--rerun-failed] [--json] <out.bin>
vox_stage2 toolchain current|list|install <vX.Y.Z>|use <vX.Y.Z>|pin <vX.Y.Z>
```

## 仓库布局

- 源码：`src/`
- 清单：`vox.toml`
- 锁文件：`vox.lock`
- 构建产物：`target/`

## 仓库门禁

```bash
make test-active   # stage2 rolling selfhost + stage2 test smoke
make test          # test-active + examples smoke
```

关键脚本：

- `scripts/ci/stage2-rolling-selfhost.sh`
- `scripts/ci/verify-stage2-p0p1.sh`

## 发布与滚动自举

- 唯一链路：`stage2(locked release) -> stage2(new)`
- 锁文件：`scripts/release/bootstrap-stage2.lock`
- 本地演练：

```bash
make release-dry-run VERSION=v0.1.0-rc1
make release-verify VERSION=v0.1.0-rc1
```

详见：`docs/24-release-process.md`。

## 历史说明

- `stage0` / `stage1` 不再保留在 `main`。
- 历史 bootstrap 代码见分支：`archive/stage0-stage1`。
