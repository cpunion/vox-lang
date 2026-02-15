# 标准库（草案）

本章用于定义 Vox 标准库的模块边界与最小可用集合。

最小集合（建议）：

- `std::prelude`：基础 trait 与常用类型导出
- `std::string`：`String`、`str`、`StrView`
- `std::fs`：最小文件系统能力（读写文件、枚举源文件；用于 编译器工具链自举）
- `std::process`：最小进程能力（args/exec/exe_path/getenv；用于 编译器工具链自举）
- `std::time`：最小时钟能力（`now_ns`；用于测试与工具链计时）
- `std::sync`：并发原语（`Mutex[T]/Atomic[T]` 泛型 API）
- `std::async`：pull 模型异步核心（`Poll[T]`、`Future`、`Context`、`Waker`）
- `std::collections`：`Vec`、`Map` 等
- `std::io`：输出 + 最小文件抽象 + 最小 TCP 抽象

当前实现落地：

- `std::prelude` 已提供默认 trait：`Eq`、`Ord`、`Show`、`Clone`、`Release`、`Into`（用于 `Result` 的 `?` 传播时 `Err` 转换）。
- 未显式 import 时，函数名会回退到 `std/prelude`；trait 静态调用与 `impl Trait for ...` 也支持回退到 `std/prelude` 的公开 trait。
- `std::string` 已提供 `StrView`（拥有型字符串视图）。
  - 基础 API：`view_all`、`view_range`、`sub`、`len`、`is_empty`、`byte_at`、`to_string`。
  - view-first 子串 API（推荐）：`take_prefix`、`take_suffix`、`drop_prefix`、`drop_suffix`。
  - 匹配/查找 API：
    - `String` 参数版本：`starts_with`、`ends_with`、`contains`、`index_of`、`last_index_of`、`equals`、`compare`
    - `StrView` 参数版本：`starts_with_view`、`ends_with_view`、`contains_view`、`index_of_view`、`last_index_of_view`、`equals_view`、`compare_view`
  - 语言层当前不支持裸 `str`（会报错）；请使用 `String`（拥有）或 `&str`/`&'static str`（借用）。同时支持 `&T` / `&mut T` / `&'static T` / `&'static mut T` 语法（借用形状在类型系统/IR 中保留为 `Ref`，`&'a T` 这类命名 lifetime 在 parser 阶段拒绝）。
  - 显式释放基线：`release(s: String) -> String`，返回空字符串并断开当前值（不触发别名 UAF）。该调用必须接收返回值（例如 `s = release(s)`），裸表达式语句会报错；被释放变量后续读取会报 `use of moved value`（除非先重绑定）。
- `std::collections` 已提供 `Slice[T]`（拥有型 `Vec[T]` 视图）。
  - 基础 API：`view_all`、`view_range`、`sub`、`len`、`is_empty`、`get`、`to_vec`。
  - view-first 子切片 API（推荐）：`take_prefix`、`take_suffix`、`drop_prefix`、`drop_suffix`。
  - 查找与匹配 API：`contains`、`index_of`、`last_index_of`、`starts_with`、`ends_with`、`contains_slice`、`index_of_slice`、`last_index_of_slice`（相关 API 需要 `T: Eq`）。
  - 比较 API：`equals`/`equals_vec`（`T: Eq`）、`compare`/`compare_vec`（`T: Ord`）。
  - 显式释放基线：`release_vec[T](v: Vec[T]) -> Vec[T]`，返回新的空 `Vec`，不释放共享底层存储（避免别名 UAF）。该调用必须接收返回值；被释放变量后续读取会报 `use of moved value`（除非先重绑定）。
- `std::collections` 还提供最小泛型 `Map[K,V]`（线性实现）：
  - 构造函数：`map[K,V]()`
  - inherent impl（`impl[K: Eq, V] Map[K,V]`）方法：
    - `len`、`is_empty`、`index_of_key`、`contains_key`
    - `get`、`get_or`（缺失键时返回调用方提供的 fallback）
    - `keys`、`values`（按当前存储顺序返回拷贝）
    - `set`（存在则覆盖，不存在则插入）、`remove`、`clear`、`release`（需接收返回值，释放后旧变量读取会报 `use of moved value`）
  - 其中键比较相关 API 需要 `K: Eq`。
  - 另外提供 `impl[K: Eq + Clone, V: Clone] Clone for Map[K,V]`（深拷贝 keys/vals，不共享底层 Vec 存储）。
  - 另外提供 `impl[K: Eq, V] Release for Map[K,V]`（返回空 `Map`，显式断开当前值）。
- `std::fs` / `std::process` 已提供最小工具链内建封装（文件读写、路径存在性、`mkdir -p`、`.vox` 枚举、命令执行、参数读取、环境变量读取）。
- `std::time` 已提供 `now_ns() -> i64`（wall-clock 纳秒时间戳，解释器与 C 后端均可用）。
- `std::io` 已提供：`out`、`out_ln`、`fail`，以及 `File`/`file_read_all`/`file_write_all`/`file_exists`/`mkdir_p`。网络部分提供 `NetAddr` + `NetConn` 与最小 TCP API：`net_connect` / `net_send` / `net_recv` / `net_close`（解释器与 C 后端一致可用；失败时统一 panic）。
- `std::sync` 当前 API 为泛型句柄：`Mutex[T]` / `Atomic[T]`（当前 `T` 由 `SyncScalar` 约束，已覆盖 `i32/i64`）；底层统一基于 runtime `i64` handle intrinsic。`fetch_add/swap/load/store` 已在解释器与 C 后端对齐。

重要：由于 Vox 的“临时借用”规则，标准库应优先提供“可长期保存的拥有型视图”（例如 `StrView` / `Slice[T]`），并优先使用 view-first API（避免不必要的 `to_string` / `to_vec` 物化）。
