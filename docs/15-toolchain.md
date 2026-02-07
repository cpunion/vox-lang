# 工具链（草案）

主命令：`vox`

```bash
vox build
vox build --release

vox run
vox ir
vox test

vox fmt
vox lint
vox doc
```

输出目录（草案）：`target/`

```text
target/
  debug/
  release/
  doc/
```

panic 策略（草案）：

- 默认：unwind（调用 drop）
- 可选：`--panic=abort`

## Stage0 产物（当前实现）

stage0 的 `vox build` 会在包根目录的 `target/debug/` 下产出：

- `<pkg>.ir`：IR v0 文本（见 `docs/19-ir-spec.md`）
- `<pkg>.c`：后端生成的 C（阶段性后端）
- `<pkg>`：可执行文件
