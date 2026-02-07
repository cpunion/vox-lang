# 标准库（草案）

本章用于定义 Vox 标准库的模块边界与最小可用集合。

最小集合（建议）：

- `std::prelude`：基础 trait 与常用类型导出
- `std::string`：`String`、`str`、`StrView`
- `std::sync`：`Arc[T]`、`Weak[T]`、（后续）`Mutex[T]`、`Atomic[T]`
- `std::collections`：`Vec`、`Map` 等
- `std::io`：文件/网络抽象（deferred，需与 effect/资源系统协调）

重要：由于 Vox 的“临时借用”规则，标准库应优先提供“可长期保存的拥有型视图”（例如 `StrView`），避免 API 设计依赖 `&T` 返回值。
