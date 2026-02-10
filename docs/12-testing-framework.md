# 测试框架（Stage0：已实现的最小子集）

目标：借鉴 `go test` 的优点，让测试像普通代码一样编译/运行，保持零运行时开销、最小样板。

## 1. 发现规则（Go 风格）

- 测试文件：
  - `src/**/*_test.vox`（与源码同包，可访问私有符号）
  - `tests/**/*.vox`（集成测试：视为 `tests` 模块，只能访问 `pub` API，类似 Go 的外部测试包）
- 测试函数：
  - 名称以 `test_` 开头
  - 必须是 `fn test_xxx() -> ()`（无参数，返回 unit）

CLI（见 `docs/15-toolchain.md`）：

```bash
vox test
vox test --engine=c
vox test --engine=interp
vox test --run='regex'
vox test --jobs=4
vox test --rerun-failed
vox test --list
vox test --json
```

Stage0 行为：

- `vox build/run`：会忽略 `src/**/*_test.vox`
- `vox test`：会包含 `src/**/*.vox` + `src/**/*_test.vox` + `tests/**/*.vox`，并执行所有 `test_*` 函数
- 依赖包（path deps）的测试文件不会被加入（保持依赖纯净）
 - `vox test --engine=c`（默认）：编译生成测试可执行文件并执行（更贴近最终语义）
 - `vox test --engine=interp`：解释执行测试（用于快速对照；能力可能更受限）
 - `vox test --run=<regex>`：仅运行匹配的测试（匹配限定名 `mod::test_x` 与短名 `test_x`）
 - `vox test --jobs=N`（或 `-j N`）：设置模块级并行度；默认 `GOMAXPROCS`，始终保持“模块内串行”
 - `vox test --rerun-failed`：仅重跑上次失败测试（缓存文件：`target/debug/.vox_failed_tests`）
 - `vox test --list`：仅列出当前筛选后的测试，不执行
- `vox test --json`：输出机器可读 JSON 报告（包含 selection/results/modules/module_details/failed_tests/slowest/summary）
 - 调度策略：模块间并行、模块内串行（稳定日志顺序与共享状态可控）
 - 输出包含每个测试耗时（如 `([OK] mod::test_x (0.42ms))`）
 - 输出包含模块级汇总（`[module] <mod>: <passed> passed, <failed> failed (<dur>)`）
- 输出包含最慢测试 TopN（当前 N=5，`[slowest] <test> (<dur>)`）
- 输出包含选择摘要（`[select] discovered: N, selected: M`，并在使用 `--run/--rerun-failed` 时打印过滤来源）
- 测试失败后输出重跑提示（`[hint] rerun failed: vox test --engine=... --rerun-failed <dir>`）
- `target/debug/.vox_failed_tests` 当前采用 JSON 元数据格式（包含失败测试列表与更新时间）；读取端兼容旧版纯文本行列表

`--json` 的关键字段：

- `selection`：发现数、筛选后数量、`--run/--jobs/--rerun-failed` 元信息
- `results`：逐测试结果（状态、耗时、错误）
- `modules`：模块级汇总（passed/failed/duration）
- `module_details`：模块级测试清单（`tests`）与失败子集（`failed_tests`）
- `failed_tests`：全局失败测试名列表（便于外部工具做后续重跑/分片）

## 2. 断言（Stage0）

Stage0 仅内建两个最底层函数：

```vox
panic(msg: String);
print(msg: String);
```

断言与测试工具由标准库提供（以 `.vox` 源码形式随 stage0 一起注入）：

- `std/prelude`：`assert` / `assert_with` / `assert_eq[T: Eq]` / `assert_ne[T: Eq]` / `assert_lt[T: Ord]` / `assert_le[T: Ord]` / `assert_gt[T: Ord]` / `assert_ge[T: Ord]` / `fail`
- `std/testing`：对 `std/prelude` 的薄封装，便于显式使用 `t.assert(...)` 这类风格

```vox
import "std/testing" as t

fn test_ok() -> () {
  t.assert(true);
  t.assert_with(1 + 1 == 2, "math broken");
  t.assert_eq(1 + 1, 2);
  t.assert_ne(1 + 1, 3);
  t.assert_eq(true, true);
  t.assert_eq("a", "a");
  // t.fail("message"); // 直接失败并打印消息
}
```

说明：

- `assert_eq/assert_ne` 是泛型函数（stage0 支持最小子集的泛型单态化 + 推导，语法支持 `T: Eq`）。
- `assert_eq` 依赖 `!=`、`assert_ne` 依赖 `==`；可用类型由当前阶段 `Eq` 与比较 lowering 覆盖范围决定。
