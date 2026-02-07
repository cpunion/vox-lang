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

## 执行引擎（Stage0：已实现）

Stage0 提供两套执行/验证引擎，便于在“快（解释器）”与“准（后端编译）”之间切换：

- `--engine=c|interp`：选择引擎（默认：`c`）
- `--compile`：`--engine=c` 的别名
- `--interp`：`--engine=interp` 的别名

支持范围：

- `vox run`：
  - `--engine=c`：IR -> C -> 调用系统 C 编译器产出可执行文件并运行
  - `--engine=interp`：用解释器直接运行
  - `vox run <dir> -- <args...>`：把 `--` 之后的参数透传给产出的程序（两种引擎一致）
- `vox test`：
  - `--engine=c`：编译生成测试可执行文件并运行（默认）
  - `--engine=interp`：用解释器运行测试（用于快速对照；能力可能更受限）
- `vox build`：
  - `--engine=c`：产出 C 与可执行文件（见下方“Stage0 产物”）
  - `--engine=interp`：仅加载/解析/类型检查（不产出二进制）

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
