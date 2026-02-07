# 测试框架（Stage0：已实现的最小子集）

目标：借鉴 `go test` 的优点，让测试像普通代码一样编译/运行，保持零运行时开销、最小样板。

## 1. 发现规则（Go 风格）

- 测试文件：
  - `src/**/*_test.vox`（与源码同包，可访问私有符号）
  - `tests/**/*.vox`（集成测试，仍按同一包编译执行；stage0 先不区分“集成/单元”的可见性）
- 测试函数：
  - 名称以 `test_` 开头
  - 必须是 `fn test_xxx() -> ()`（无参数，返回 unit）

CLI（见 `docs/15-toolchain.md`）：

```bash
vox test
```

Stage0 行为：

- `vox build/run`：会忽略 `src/**/*_test.vox`
- `vox test`：会包含 `src/**/*.vox` + `src/**/*_test.vox` + `tests/**/*.vox`，并执行所有 `test_*` 函数
- 依赖包（path deps）的测试文件不会被加入（保持依赖纯净）

## 2. 断言（Stage0）

Stage0 内建：

```vox
assert(cond);
```

同时提供一个最小的 `std/testing`（目前是编译器内建模块，而非真实源码文件）：

```vox
import "std/testing" as t

fn test_ok() -> () {
  t.assert(true);
  t.assert_eq_i32(1 + 1, 2);
  t.assert_eq_i64(1, 1);
  t.assert_eq_bool(true, true);
  t.assert_eq_str("a", "a");
  // t.fail("message");
}
```

说明：

- 由于 stage0 暂无宏/泛型/重载，`assert_eq` 暂时以 `assert_eq_i32/i64/bool/str` 的形式提供。
- 后续引入宏/泛型后，会把这套 API 收敛成更自然的 `assert_eq!(a, b)` 或 `assert_eq(a, b)`。
