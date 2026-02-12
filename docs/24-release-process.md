# Release Process（版本化发布 + 锁定版滚动自举）

本章定义 `vox-lang` 的发布策略：

- 首个公开版本：`v0.1.0`
- 产物覆盖：`stage0` / `stage1` / `stage2`
- 平台覆盖：`linux` / `darwin` / `windows`
- 后续升级：采用“**锁定版本的 stage2 二进制**”滚动构建新的 stage2

## 1. 发布产物

每个版本发布以下平台包：

- `vox-lang-<version>-linux-amd64.tar.gz`
- `vox-lang-<version>-darwin-amd64.tar.gz`
- `vox-lang-<version>-windows-amd64.tar.gz`

每个包包含：

- `bootstrap/vox-stage0[.exe]`
- `bootstrap/vox-stage1[.exe]`
- `bin/vox-stage2[.exe]`
- `VERSION`
- `BOOTSTRAP_MODE`（CI 要求为 `rolling-stage2`）

并发布对应校验文件：

- `vox-lang-<version>-<platform>.tar.gz.sha256`

## 2. 构建链路

### 2.1 `v0.1.0`（冷启动）

`stage0 -> stage1(tool) -> stage2(tool)`

- `stage0` 由 Go 直接构建。
- `stage1` 先由 stage0 构建，再生成 tool driver 版本。
- `stage2` 由 stage1(tool) 构建为 tool driver 版本。

### 2.2 `v0.1.1+`（锁定版滚动）

优先链路：

`stage2(locked release) -> stage2(new)`

本地应急链路（仅手工场景）：

`stage1(tool) -> stage2(new)`

## 3. 锁文件与策略

锁文件：`scripts/release/bootstrap-stage2.lock`

当前字段：

- `STAGE2_BOOTSTRAP_TAG`：锁定用于滚动自举的 release tag
- `ALLOW_STAGE1_FALLBACK`：锁定二进制不可用时是否允许回退 stage1

CI 步骤：

- `scripts/release/prepare-locked-bootstrap.sh <repo> <platform>`
- 尝试下载锁定资产：`vox-lang-${STAGE2_BOOTSTRAP_TAG}-${platform}.tar.gz`
- 成功则提取 `bin/vox-stage2[.exe]` 到 `compiler/stage2/target/bootstrap/vox_stage2_prev[.exe]`
- 失败时按 `ALLOW_STAGE1_FALLBACK` 决定“回退继续”或“直接失败”

当前 CI 策略：

1. `ALLOW_STAGE1_FALLBACK=false`（锁定资产不可用即失败）
2. `VOX_REQUIRE_ROLLING_BOOTSTRAP=1`（禁止在 CI 中回退到 stage1）

## 4. 触发规则

- CI 构建校验：`pull_request -> main`（三平台构建 + 烟测，不发布 release）
- 版本发布：`push tag vX.Y.Z`（三平台构建 + 烟测 + GitHub Release）
- 推荐流程：
  1. PR 合并到 `main`（已完成三平台构建与烟测）
  2. 打 tag
  3. GitHub Actions 上传 release 资产

## 5. 验收门禁

发布 workflow 至少满足：

1. 三个平台均成功产出 `stage0/stage1/stage2` 二进制。
2. 三个平台 `BOOTSTRAP_MODE` 均为 `rolling-stage2`。
3. 每个平台产物均产出 `.sha256`。
4. `scripts/release/verify-release-bundle.sh` 对每个平台产物验证通过。
5. tag 发布时 upload 全量资产到 GitHub Release。

## 6. 锁版本维护流程

发布 `vX.Y.Z` 成功后，建议更新锁文件：

1. 把 `STAGE2_BOOTSTRAP_TAG` 更新为 `vX.Y.Z`
2. 提交到 `main`
3. 后续版本继续基于该锁定版本滚动

这一步使“下一次发布使用哪个 stage2 自举”明确可审计，避免“自动选择最近 release”带来的不确定性。

## 7. 本地演练

推荐直接执行单命令 dry-run（构建 + 烟测 + 产物校验）：

```bash
make release-dry-run VERSION=v0.1.0-rc1
```

等价脚本链路：

```bash
./scripts/release/dry-run-rolling.sh v0.1.0-rc1
```

若只做产物结构校验：

```bash
make release-verify VERSION=v0.1.0-rc1
```

输出在 `dist/` 下。
