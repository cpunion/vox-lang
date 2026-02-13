# 自举与阶段策略（当前）

## 结论

仓库主线已收敛为单编译器实现：

- `main` 只保留一个编译器实现（位于 `src/`）。
- 日常自举链路为：`locked release compiler -> new compiler`。
- `stage0/stage1` 历史实现不再在 `main` 维护，统一归档到分支 `archive/stage0-stage1`。

## 主线目录约定

- `src/`：编译器与标准库源码
- `vox.toml`：主包清单
- `vox.lock`：依赖锁文件
- `target/`：构建输出与自举缓存

## 门禁

- `make test-active`：rolling selfhost + test smoke
- `make test`：在 `test-active` 基础上增加 examples smoke

## 发布与锁定自举

- 锁文件：`scripts/release/bootstrap.lock`（将来会去掉历史命名）
- 下载锁定版：`scripts/release/prepare-locked-bootstrap.sh`
- 产物构建：`scripts/release/build-release-bundle.sh`
- 产物校验：`scripts/release/verify-release-bundle.sh`

详见：`docs/24-release-process.md`。
