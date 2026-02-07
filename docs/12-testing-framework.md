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
```

Stage0 行为：

- `vox build/run`：会忽略 `src/**/*_test.vox`
- `vox test`：会包含 `src/**/*.vox` + `src/**/*_test.vox` + `tests/**/*.vox`，并执行所有 `test_*` 函数
- 依赖包（path deps）的测试文件不会被加入（保持依赖纯净）
 - `vox test --engine=c`（默认）：编译生成测试可执行文件并执行（更贴近最终语义）
 - `vox test --engine=interp`：解释执行测试（用于快速对照；能力可能更受限）

## 2. 断言（Stage0）

Stage0 仅内建两个最底层函数：

```vox
panic(msg: String);
print(msg: String);
```

断言与测试工具由标准库提供（以 `.vox` 源码形式随 stage0 一起注入）：

- `std/prelude`：`assert` / `assert_eq[T]` / `fail`
- `std/testing`：对 `std/prelude` 的薄封装，便于显式使用 `t.assert(...)` 这类风格

```vox
import "std/testing" as t

fn test_ok() -> () {
  t.assert(true);
  t.assert_eq(1 + 1, 2);
  t.assert_eq(true, true);
  t.assert_eq("a", "a");
  // t.fail("message"); // 直接失败并打印消息
}
```

说明：

- `assert_eq` 是泛型函数（stage0 支持最小子集的泛型单态化 + 推导）。
- 目前 `assert_eq` 依赖 `!=`：因此在 stage0 里只支持 `bool/i32/i64/String` 的比较（其他类型后续再做 lowering/trait 化）。
