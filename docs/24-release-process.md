# Release Process（版本化发布 + 二进制滚动）

本章定义 `vox-lang` 的发布策略：

- 首个公开版本：`v0.1.0`
- 产物覆盖：`stage0` / `stage1` / `stage2`
- 平台覆盖：`linux` / `darwin` / `windows`
- 后续升级：优先使用“上一版 stage2 二进制”滚动构建新的 stage2

## 1. 发布产物

每个版本发布以下平台包：

- `vox-lang-<version>-linux-amd64.tar.gz`
- `vox-lang-<version>-darwin-amd64.tar.gz`
- `vox-lang-<version>-windows-amd64.tar.gz`

每个包包含：

- `bin/vox-stage0[.exe]`
- `bin/vox-stage1[.exe]`
- `bin/vox-stage2[.exe]`
- `VERSION`

并发布对应校验文件：

- `vox-lang-<version>-<platform>.tar.gz.sha256`

## 2. 构建链路

### 2.1 `v0.1.0`（冷启动）

`stage0 -> stage1(tool) -> stage2(tool)`

- `stage0` 由 Go 直接构建。
- `stage1` 先由 stage0 构建，再生成 tool driver 版本。
- `stage2` 由 stage1(tool) 构建为 tool driver 版本。

### 2.2 `v0.1.1+`（滚动）

优先链路：

`stage2(prev release) -> stage2(new)`

回退链路：

`stage1(tool) -> stage2(new)`

规则：

1. 如果找到并可执行上一版 stage2 二进制，则优先使用它构建新 stage2。
2. 若上一版不可用或构建失败，自动回退到 stage1(tool) 构建。
3. 回退属于发布日志中需要记录的事件。

## 3. 触发规则

- CI 构建校验：`pull_request -> main`（三平台构建 + 烟测，不发布 release）
- 版本发布：`push tag vX.Y.Z`（三平台构建 + 烟测 + GitHub Release）
- 推荐流程：
  1. PR 合并到 `main`（已完成三平台构建与烟测）
  2. 打 tag
  3. GitHub Actions 上传 release 资产

## 4. 验收门禁

发布 workflow 至少满足：

1. 三个平台均成功产出 `stage0/stage1/stage2` 二进制。
2. 每个平台产物均产出 `.sha256`。
3. release 上传所有平台资产。

## 5. 本地演练

本地可先执行：

```bash
./scripts/release/build-release-bundle.sh v0.1.0
```

输出在 `dist/` 下。
