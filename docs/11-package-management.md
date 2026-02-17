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
- `dep = { git = "https://...", rev = "..." }`
- `dep = { git = "../local/repo", version = "tag-or-branch" }`
- `dep = "1.2.3"`（从本地 registry cache 解析）

## 目录约定（草案）

```
my_app/
  vox.toml
  pkgs/        # optional reusable packages workspace
  src/
    main.vox   # 可执行入口
  tests/
  examples/
  target/
```

## 依赖来源

Stage1 当前能力：

- `path`：支持，且支持递归加载传递依赖
- `git`：支持 clone/fetch/checkout（支持本地路径与远程 URL）
- `registry`：支持从本地 cache 目录解析 `name/version`（默认 `.vox/deps/registry`，可通过 `registry` 字段覆盖根目录）；Stage2 额外支持 `registry = "git+..."`（或 URL/`.git` 形式）自动 clone/fetch 到 `.vox/deps/registry_remote` 后再解析 `name/version`。

## 锁文件：`vox.lock`

Stage2 在 `build` / `test` 成功解析依赖后会写出 `vox.lock`，并在后续构建时做完整性校验。当前字段包含：

- `name`
- `source`（`path` / `git` / `registry`）
- `path`（声明式 path 依赖）
- `resolved_path`（解析后的本地目录）
- `git` / `rev`（git 依赖会记录解析后的具体 commit）
- `registry` / `version`
- `digest`（依赖源码快照摘要，基于 `vox.toml` + `src/**/*.vox` 的稳定哈希）

完整性规则（stage1 当前实现）：

- 若存在 `vox.lock`，构建前会校验每个依赖条目的 `source/path/git/rev/registry/version/digest` 与当前解析结果一致。
- 不一致会直接失败，并给出字段级诊断，例如：
  - `dependency mismatch: dep field digest expected="..." actual="..."`
  - `missing dependency in vox.lock: dep`
  - `unexpected dependency in vox.lock: dep`
  - `dependency count mismatch: lock=1 resolved=2`
- `build` 与 `test` 会输出一致 remediation hint：
  - `hint: refresh lockfile after dependency changes: remove vox.lock then rerun build/test.`
- 这样可以防止“依赖内容悄悄变化但版本号不变”的非可复现构建。
