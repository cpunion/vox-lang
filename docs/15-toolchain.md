# 工具链（草案）

主命令：`vox`

```bash
vox build
vox build --release

vox run
vox ir
vox c
vox test
vox test --run='regex'
vox test --rerun-failed
vox test --list
vox test --json

vox fmt
vox lint
vox doc
```

仓库开发辅助（Makefile）：

```bash
make test
make audit-vox-lines
make audit-vox-lines MAX=180
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
  - `--run=<regex>`：按正则过滤测试名（支持限定名或短名匹配）
  - `--rerun-failed`：仅运行上次失败测试（读取 `target/debug/.vox_failed_tests`）
  - `--list`：仅列出筛选后的测试并退出（不执行）
  - `--json`：输出 JSON 测试报告（方便 CI/工具消费）
- `vox build`：
  - `--engine=c`：产出 C 与可执行文件（见下方“Stage0 产物”）
  - `--engine=interp`：仅加载/解析/类型检查（不产出二进制）
- `vox fmt`：
  - 递归格式化 `src/**/*.vox` 与 `tests/**/*.vox`
  - 当前规则为“最小可逆格式化”：去除行尾空白、统一换行为 `\n`、确保文件末尾单个换行
- `vox lint`：
  - 执行包级语义检查（等价于 build 的加载/解析/类型检查，不产出二进制）
  - 额外输出文本告警（当前包含超长行提示）
- `vox doc`：
  - 从包源码生成最小 API 文档到 `target/doc/API.md`
  - 当前覆盖公开 `type/const/struct/enum/trait/fn` 符号（按模块分组）

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

另：`vox c [dir]` 会把当前 stage0 生成的 C 直接打印到 stdout（用于快速审阅后端输出）。

## Stage1 编译器 CLI（自举目标）

Stage1 编译器是用 Vox 写的编译器（位于 `compiler/stage1`），其二进制通常由 Stage0 产出：

- `compiler/stage1/target/debug/vox_stage1`

当前 Stage1 CLI（以代码为准，见 `compiler/stage1/src/main.vox`）：

```text
vox_stage1 emit-c   [--driver=user|tool] <out.c>   <src...>
vox_stage1 build    [--driver=user|tool] <out.bin> <src...>
vox_stage1 build-pkg [--driver=user|tool] <out.bin>   # 从当前目录 ./src 自动发现源码
vox_stage1 test-pkg  <out.bin>   # 从 ./src 与 ./tests 发现并运行 test_*
```

说明（当前实现）：

- `--driver=user`（默认）：为被编译出的二进制生成“用户 driver main”，会打印 `main()` 的返回值到 stdout（便于最小 demo）。
- `--driver=tool`：为被编译出的二进制生成“工具 driver main”，不打印返回值，并把 `main() -> i32` 作为进程退出码返回（用于编译器/工具链自举）。
- `build-pkg/test-pkg` 会读取当前目录的 `vox.toml`，加载其中声明的 path 依赖（包含传递依赖；依赖只加载其 `src/**`，不加载 tests）。
  - `vox.toml` 无效或出现重复依赖名时，命令会失败并返回非 0。
- Stage1 会按可执行文件路径推导 Stage1 根目录，并从其 `src/std/**` 注入标准库源码（用于自举期最小 std）。
