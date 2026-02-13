# 线程安全（当前基线）

当前编译器采用 `Send`/`Sync` marker trait 作为线程安全约束入口：

- 位置：`std/prelude`。
- 形态：marker trait（无方法）。
- 用法：泛型约束（如 `fn f[T: Send](x: T) -> ()`）。

自动推导规则（typecheck 层）：

1. 基础类型自动满足：`()`、`bool`、整数、浮点、`String`。
2. `Range[T]`、`Vec[T]`：当 `T` 满足对应 trait 时自动满足。
3. `struct`、`enum`：当所有字段（含变体载荷）都满足时自动满足。
4. 类型参数 `T` 不做隐式推导：`T: Send/Sync` 必须显式写在泛型约束里。
5. 若存在显式 `impl Send/Sync for X`，显式实现优先；否则走自动推导。

当前边界：

- 还不支持 negative impl（如 `!Send`）与 unsafe 手工承诺模型。
- 更深层的借用/所有权收敛与并发内存模型仍在后续阶段推进（见 `docs/27-active-backlog.md` 的 `D06/D07`）。
