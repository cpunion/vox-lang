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

## Stage1 编译器 CLI（自举目标）

Stage1 编译器是用 Vox 写的编译器（位于 `compiler/stage1`），其二进制通常由 Stage0 产出：

- `compiler/stage1/target/debug/vox_stage1`

当前 Stage1 CLI（以代码为准，见 `compiler/stage1/src/main.vox`）：

```text
vox_stage1 emit-c   <out.c>   <src...>
vox_stage1 build    <out.bin> <src...>
vox_stage1 build-pkg <out.bin>   # 从当前目录 ./src 自动发现源码
vox_stage1 test-pkg  <out.bin>   # 从 ./src 与 ./tests 发现并运行 test_*
```

说明（当前实现）：

- `build-pkg/test-pkg` 会读取当前目录的 `vox.toml`，加载其中声明的 path 依赖（依赖只加载其 `src/**`，不加载 tests）。
- Stage1 会按可执行文件路径推导 Stage1 根目录，并从其 `src/std/**` 注入标准库源码（用于自举期最小 std）。
