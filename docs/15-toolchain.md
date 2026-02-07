# 工具链（草案）

主命令：`vox`

```bash
vox build
vox build --release

vox run
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

