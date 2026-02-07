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

## 目录约定（草案）

```
my_app/
  vox.toml
  src/
    main.vox   # 可执行入口
    lib.vox    # 库入口（可选）
  tests/
  examples/
  target/
```

## 依赖来源（deferred）

- registry
- git
- path

## 锁文件（deferred）

是否引入 `vox.lock`、其格式与可复现策略待定。

