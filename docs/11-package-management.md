# 包管理（草案）

## 清单文件：`vox.toml`

```toml
[package]
name = "my_app"
version = "0.1.0"
edition = "2026"

[dependencies]
http = "1.0"
json = { version = "2.0", features = ["serde"] }
```

Stage1 当前可解析：

- `dep = { path = "../dep" }`
- `dep = { path = "../dep", version = "0.1.0" }`
- `dep = "1.2.3"`（仅解析；Stage1 目前不支持非 path 拉取）

## 目录约定（草案）

```
my_app/
  vox.toml
  src/
    main.vox   # 可执行入口
  tests/
  examples/
  target/
```

## 依赖来源

Stage1 当前能力：

- `path`：支持，且支持递归加载传递依赖
- `registry`：未实现（遇到非 path 依赖会报错）
- `git`：未实现

## 锁文件：`vox.lock`

Stage1 在 `build-pkg` / `test-pkg` 成功解析依赖后会写出 `vox.lock`（当前为最小实现，记录依赖名与已解析 path/version 字段）。
