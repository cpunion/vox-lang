# Release Process（版本化发布 + 锁定版 stage2 滚动自举）

本章定义 `vox-lang` 当前发布策略：

- 发布产物：`stage2`（单一可执行）
- 平台覆盖：`linux` / `darwin` / `windows`
- 构建链路：`stage2(locked release) -> stage2(new)`
- 禁止链路：发布与 CI 门禁不再使用 `stage1` fallback

## 1. 发布产物

每个版本发布以下平台包：

- `vox-lang-<version>-linux-amd64.tar.gz`
- `vox-lang-<version>-darwin-amd64.tar.gz`
- `vox-lang-<version>-windows-amd64.tar.gz`

每个包包含：

- `bin/vox-stage2[.exe]`
- `VERSION`
- `BOOTSTRAP_MODE`（必须为 `rolling-stage2`）

并发布对应校验文件：

- `vox-lang-<version>-<platform>.tar.gz.sha256`

## 2. 构建链路

唯一链路：

`stage2(locked release binary) -> stage2(new)`

要求：

- 必须提供 rolling bootstrap 二进制（`VOX_BOOTSTRAP_STAGE2` 或 `compiler/stage2/target/bootstrap/vox_stage2_prev`）。
- 缺失 bootstrap 二进制时构建直接失败，不允许回退到 `stage1`。

## 3. 锁文件

锁文件：`scripts/release/bootstrap-stage2.lock`

当前字段：

- `STAGE2_BOOTSTRAP_TAG`：锁定用于滚动自举的 release tag

CI 步骤：

- `scripts/release/prepare-locked-bootstrap.sh <repo> <platform>`
- 下载锁定资产：`vox-lang-${STAGE2_BOOTSTRAP_TAG}-${platform}.tar.gz`
- 提取 `bin/vox-stage2[.exe]` 到 `compiler/stage2/target/bootstrap/vox_stage2_prev[.exe]`
- 下载/提取失败直接失败

## 4. 触发规则

- CI 构建校验：`pull_request -> main`（三平台构建 + 烟测，不发布）
- 版本发布：`push tag vX.Y.Z`（三平台构建 + 烟测 + GitHub Release）

推荐流程：

1. PR 合并到 `main`（完成三平台构建与烟测）
2. 打 tag
3. GitHub Actions 上传 release 资产

## 5. 验收门禁

发布 workflow 至少满足：

1. 三个平台均成功产出 `vox-stage2`。
2. 三个平台 `BOOTSTRAP_MODE` 均为 `rolling-stage2`。
3. 每个平台产物均产出 `.sha256`。
4. `scripts/release/verify-release-bundle.sh` 对每个平台产物验证通过。
5. tag 发布时上传全量资产到 GitHub Release。

## 6. 锁版本维护流程

发布 `vX.Y.Z` 成功后：

1. 更新 `scripts/release/bootstrap-stage2.lock` 的 `STAGE2_BOOTSTRAP_TAG=vX.Y.Z`
2. 提交到 `main`
3. 后续版本继续基于该锁定版本滚动

## 7. 本地演练

推荐执行：

```bash
make release-dry-run VERSION=v0.1.0-rc1
```

等价脚本：

```bash
./scripts/release/dry-run-rolling.sh v0.1.0-rc1
```

只做产物结构校验：

```bash
make release-verify VERSION=v0.1.0-rc1
```
