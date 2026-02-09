# 标准库（草案）

本章用于定义 Vox 标准库的模块边界与最小可用集合。

最小集合（建议）：

- `std::prelude`：基础 trait 与常用类型导出
- `std::string`：`String`、`str`、`StrView`
- `std::fs`：最小文件系统能力（读写文件、枚举源文件；用于 stage1 工具链自举）
- `std::process`：最小进程能力（args/exec；用于 stage1 工具链自举）
- `std::sync`：`Arc[T]`、`Weak[T]`、（后续）`Mutex[T]`、`Atomic[T]`
- `std::collections`：`Vec`、`Map` 等
- `std::io`：文件/网络抽象（deferred，需与 effect/资源系统协调）

当前 stage1 落地：

- `std::prelude` 已提供默认 trait：`Eq`、`Show`。
- 未显式 import 时，函数名会回退到 `std/prelude`；trait 静态调用与 `impl Trait for ...` 也支持回退到 `std/prelude` 的公开 trait。

重要：由于 Vox 的“临时借用”规则，标准库应优先提供“可长期保存的拥有型视图”（例如 `StrView`），避免 API 设计依赖 `&T` 返回值。
