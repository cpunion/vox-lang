# 标准库（草案）

本章用于定义 Vox 标准库的模块边界与最小可用集合。

最小集合（建议）：

- `std::prelude`：基础 trait 与常用类型导出
- `std::string`：`String`、`str`、`StrView`
- `std::fs`：最小文件系统能力（读写文件、枚举源文件；用于 stage1 工具链自举）
- `std::process`：最小进程能力（args/exec；用于 stage1 工具链自举）
- `std::time`：最小时钟能力（`now_ns`；用于测试与工具链计时）
- `std::sync`：并发原语（当前已落地 concrete 句柄 API，后续再泛型化）
- `std::collections`：`Vec`、`Map` 等
- `std::io`：输出 + 最小文件抽象 + 最小 TCP 抽象

当前 stage1 落地：

- `std::prelude` 已提供默认 trait：`Eq`、`Ord`、`Show`、`Into`（用于 `Result` 的 `?` 传播时 `Err` 转换）。
- 未显式 import 时，函数名会回退到 `std/prelude`；trait 静态调用与 `impl Trait for ...` 也支持回退到 `std/prelude` 的公开 trait。
- `std::fs` / `std::process` 已提供最小工具链内建封装（文件读写、路径存在性、`mkdir -p`、`.vox` 枚举、命令执行、参数读取）。
- `std::time` 已提供 `now_ns() -> i64`（wall-clock 纳秒时间戳，解释器与 C 后端均可用）。
- `std::io` 已提供：`out`、`out_ln`、`fail`，以及 `File`/`file_read_all`/`file_write_all`/`file_exists`/`mkdir_p`。网络部分提供 `NetAddr` + `NetConn` 与最小 TCP API：`net_connect` / `net_send` / `net_recv` / `net_close`（解释器与 C 后端一致可用；失败时统一 panic）。
- `std::sync` 当前提供 concrete 句柄族：`MutexI32/AtomicI32` 与 `MutexI64/AtomicI64`，统一基于 runtime `i64` handle intrinsic。`fetch_add/swap/load/store` 已在解释器与 C 后端对齐。

重要：由于 Vox 的“临时借用”规则，标准库应优先提供“可长期保存的拥有型视图”（例如 `StrView`），避免 API 设计依赖 `&T` 返回值。
