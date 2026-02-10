# 标准库（草案）

本章用于定义 Vox 标准库的模块边界与最小可用集合。

最小集合（建议）：

- `std::prelude`：基础 trait 与常用类型导出
- `std::string`：`String`、`str`、`StrView`
- `std::fs`：最小文件系统能力（读写文件、枚举源文件；用于 stage1 工具链自举）
- `std::process`：最小进程能力（args/exec；用于 stage1 工具链自举）
- `std::sync`：`Arc[T]`、`Weak[T]`、（后续）`Mutex[T]`、`Atomic[T]`
- `std::collections`：`Vec`、`Map` 等
- `std::io`：最小输出能力已落地；文件/网络抽象 deferred（需与 effect/资源系统协调）

当前 stage1 落地：

- `std::prelude` 已提供默认 trait：`Eq`、`Ord`、`Show`、`Into`（用于 `Result` 的 `?` 传播时 `Err` 转换）。
- 未显式 import 时，函数名会回退到 `std/prelude`；trait 静态调用与 `impl Trait for ...` 也支持回退到 `std/prelude` 的公开 trait。
- `std::fs` / `std::process` 已提供最小工具链内建封装（文件读写、路径存在性、`mkdir -p`、`.vox` 枚举、命令执行、参数读取）。
- `std::io` 已提供最小输出接口：`out`、`out_ln`、`fail`。
- `std::sync` 已提供 stage1 单线程占位类型：`MutexI32`、`AtomicI32`（接口先行；后续补齐可扩展的泛型版本与真实并发语义）。

重要：由于 Vox 的“临时借用”规则，标准库应优先提供“可长期保存的拥有型视图”（例如 `StrView`），避免 API 设计依赖 `&T` 返回值。
