# Release Process（版本化发布 + 锁定版编译器滚动自举）

本章定义 `vox-lang` 当前发布策略：

- 发布产物：`vox`（单一可执行）
- 平台覆盖：`linux` / `darwin` / `windows`
- 构建链路：`compiler(locked release) -> compiler(new)`
- 禁止链路：发布与 CI 门禁不再使用 `stage1` fallback

## 1. 发布产物

每个版本发布以下二进制平台包：

- `vox-lang-<version>-linux-amd64.tar.gz`
- `vox-lang-<version>-darwin-amd64.tar.gz`
- `vox-lang-<version>-windows-amd64.tar.gz`

并额外发布自制源码包：

- `vox-lang-src-<version>.tar.gz`

二进制平台包包含：

- `bin/vox[.exe]`
- `VERSION`
- `BOOTSTRAP_MODE`（必须为 `rolling`）

源码包包含仓库源码（不含 `.git`），并将 `src/vox/buildinfo/buildinfo.vox` 固定为 release 通道。

并发布对应校验文件：

- `vox-lang-<version>-<platform>.tar.gz.sha256`
- `vox-lang-src-<version>.tar.gz.sha256`

## 2. 构建链路

唯一链路：

`compiler(locked release binary) -> compiler(new)`

要求：

- 必须提供 rolling bootstrap 二进制（`VOX_BOOTSTRAP` 或 `target/bootstrap/vox_prev`）。
- 缺失 bootstrap 二进制时构建直接失败，不允许回退到 `stage1`。

## 3. 锁文件

锁文件：`scripts/release/bootstrap.lock`

当前字段：

- `BOOTSTRAP_TAG`：锁定用于滚动自举的 release tag

CI 步骤：

- `scripts/release/prepare-locked-bootstrap.sh <repo> <platform>`
- 下载锁定资产：`vox-lang-${BOOTSTRAP_TAG}-${platform}.tar.gz`
- 提取 `bin/vox[.exe]` 到 `target/bootstrap/vox_prev[.exe]`
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

1. 三个平台均成功产出 `vox`。
2. 三个平台 `BOOTSTRAP_MODE` 均为 `rolling`。
3. 每个平台产物均产出 `.sha256`。
4. `scripts/release/verify-release-bundle.sh` 对每个平台产物验证通过。
5. tag 发布时上传全量资产到 GitHub Release。
6. 产物内 `bin/vox[.exe] version` 可输出内嵌版本号。
7. 自制源码包 `vox-lang-src-<version>.tar.gz` 与其 `.sha256` 产出并通过校验。

## 6. 锁版本维护流程

发布 `vX.Y.Z` 成功后：

1. 更新 `scripts/release/bootstrap.lock` 的 `BOOTSTRAP_TAG=vX.Y.Z`
2. 提交到 `main`
3. 后续版本继续基于该锁定版本滚动

## 7. 本地演练

推荐执行：

```bash
make release-dry-run VERSION=v0.2.0-rc1
```

等价脚本：

```bash
./scripts/release/dry-run-rolling.sh v0.2.0-rc1
```

只校验二进制包：

```bash
make release-verify VERSION=v0.2.0-rc1
```

校验源码包：

```bash
make release-source-verify VERSION=v0.2.0-rc1
```
