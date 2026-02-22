# Net/IO/OS/Runtime 分层重构 Backlog（独立执行）

Status: active.

本清单是独立 backlog 文档，用于一次性推进标准库分层重构，不替代 `docs/internal/27-active-backlog.md` 的全局治理规则。

## 1. 目标与边界

目标约束（冻结）：

- `std/sys` 是最薄平台层，主要是 syscall/CRT/Winsock 级 FFI 与最小适配。
- `std/net` 承载 TCP/UDP 基础封装（client/server/peer），连接建立与收发在这里。
- `std/io` 是流抽象层（trait + bufio 风格实现），不承载连接建立语义。
- `std/os` 语义定位接近 Go `os`，承载进程/环境/系统语义封装。
- `std/runtime` 收缩为编译器必须的最小基础能力；其余能力迁出。

非目标（本批不做）：

- 不在本批改造原子语义为 IR 指令（单独 deferred backlog）。
- 不在本批引入全新网络协议栈（TLS/HTTP2/QUIC 等）。

## 2. 包职责与依赖方向

强约束依赖图（目标态）：

- `std/sys` 不依赖 `std/net/std/io/std/os/std/runtime`。
- `std/net` 依赖 `std/sys`（可选依赖 `std/io` 仅用于实现 trait，不反向依赖）。
- `std/io` 核心抽象不依赖 `std/net/std/sys`。
- `std/os` 依赖 `std/sys` 与必要基础包。
- `std/runtime` 仅依赖编译期必须基础包；不再承载业务语义入口。

禁止项（目标态）：

- `std/io` 中禁止定义 `connect/listen/accept`。
- `std/runtime` 中禁止保留 TCP/文件遍历/环境/进程执行/mutex 等业务能力入口。

## 3. 目标 API 清单

以下是目标态放置规划（`[keep]` 保留，`[move]` 迁移，`[new]` 新增，`[drop]` 删除）。

### 3.1 std/sys

Types:

- `[keep]` 无高层对象类型，保持 handle/fd + 基本标量。

Functions:

- `[keep]` `open_read/read/write/close/creat/access/mkdir/system/calloc/free`
- `[keep/new]` `connect/listen/accept/send/recv/close_socket/wait_read/wait_write`
- `[drop]` 不保留高层连接对象与业务语义。

### 3.2 std/net

Types:

- `[keep]` `NetProto`
- `[keep]` `SocketAddr`
- `[move]` `NetConn`（从 `std/io` 迁入）
- `[new]` `TcpListener`
- `[keep]` `UdpSocket`（占位到可执行）
- `[keep/move]` 网络结果类型（`NetI32Result/NetStringResult/NetBoolResult/NetCloseResult`）

Functions:

- `[keep]` `socket_addr/tcp_addr/udp_addr/parse_socket_uri`

Methods:

- `[keep/new]` `SocketAddr.uri/tcp_connect/listen/bind_udp`
- `[keep/move]` `NetConn.send/recv/wait_read/wait_write/close` + `try_*`
- `[new]` `TcpListener.accept/close`
- `[keep]` `UdpSocket.send_to/recv_from`（先保留占位，后续补真实实现）

### 3.3 std/io

Traits:

- `[new]` `Reader`
- `[new]` `Writer`
- `[new]` `Closer`
- `[new]` `ReadWriter`

Types:

- `[new]` `BufReader`
- `[new]` `BufWriter`

Functions:

- `[new]` `copy/read_all/write_all`（按可实现性分批落地）
- `[keep]` `out/out_ln/fail`（兼容期保留）
- `[drop]` 移除或转发 `io` 层网络连接建立入口。

### 3.4 std/os

Types:

- `[keep/new]` 进程/环境语义类型（按需求最小化）。

Functions:

- `[move]` `args/exe_path/getenv`
- `[move/keep]` `exec`
- `[keep or staged]` `read_file/walk_files`（若暂无法纯 `sys` 化则保留在 `os`）

### 3.5 std/runtime

Keep（最小基础）：

- `[keep]` `intrinsic_abi/has_intrinsic`
- `[keep]` `wake_notify/wake_wait/wake_wait_any`
- `[keep]` `atomic_i32_*`、`atomic_i64_*`（暂留函数路径）

Drop（迁出）：

- `[drop]` `args/exe_path/getenv`
- `[drop]` `now_ns/yield_now`
- `[drop]` `exec/read_file/walk_files`
- `[drop]` `tcp_*`
- `[drop]` `mutex_i32_*`、`mutex_i64_*`

## 4. 分批执行 Backlog（带稳定 ID）

### P0 分层冻结与骨架

- [ ] NIO-00 边界冻结
  - [ ] 文档明确各包职责、允许依赖和禁止项。
  - [ ] `docs/internal/13-standard-library.md` 同步分层说明。

### P1 sys 与 net 归位

- [ ] NIO-01 `std/sys` 网络最薄接口统一
  - [ ] 统一 `connect/listen/accept/send/recv/close_socket/wait_read/wait_write` 入口。
  - [ ] linux/darwin/windows/wasm 分支补齐，无法支持项明确 panic/stub 语义。
  - [ ] `std/sys` 测试补齐接口 smoke 与失败语义。

- [x] NIO-02 `std/net` 承载连接生命周期
  - [x] `NetConn`/连接方法迁入并在 `net` 作为主入口。
  - [x] `SocketAddr.tcp_connect/listen` 接入 `sys.*`。
  - [x] `http_*` 与 `Client` 调用链不再依赖 `io.connect` 风格入口。

### P2 io 抽象化

- [ ] NIO-03 `std/io` 去网络职责
  - [x] 删除或转发连接建立 API，保留仅兼容包装。
  - [x] 新增 `Reader/Writer/Closer/ReadWriter` trait 基线。
  - [ ] `BufReader/BufWriter` 与 `copy/read_all/write_all` 分批落地。

### P3 os 与 runtime 收缩

- [x] NIO-04 `std/os` 收拢进程/环境语义
  - [x] `args/exe_path/getenv/exec` 从 `runtime` 迁入 `os`。
  - [x] 文件语义（`read_file/walk_files`）按可实现性决定留 `os` 还是继续下沉 `sys`。

- [x] NIO-05 `std/runtime` 缩减到最小基础
  - [x] 移除 tcp/mutex/os 语义 API。
  - [x] 仅保留 wake/atomic/intrinsic probe 等编译器基础能力。
  - [x] `std/async` 路径持续可用（wake 相关不退化）。

### P4 兼容层与门禁

- [ ] NIO-06 兼容包装与淘汰计划
  - [ ] 对外兼容入口保留一阶段并打上迁移注释。
  - [ ] 下一阶段清理兼容层，避免长期双入口。

- [ ] NIO-07 测试与 CI 门禁
  - [x] `std/sys/std/net/std/io/std/os/std/runtime` 单测补齐。
  - [ ] `vox/typecheck` 与 `vox/compile` smoke 同步。
  - [x] 如有 gate 规则（例如 `vox_*` FFI 使用范围）按新分层更新并补回归。

## 5. 发布/Bootstrap 决策矩阵

默认结论：本重构应优先走“无滚动发布”路径。

不需要滚动发布（目标）：

- 仅做 std 分层与调用迁移。
- 不新增 intrinsic。
- 不新增 bootstrap 旧编译器无法链接的必须符号。

需要滚动发布（触发条件）：

- 新增 intrinsic 或改变 intrinsic 语义。
- 新增 bootstrap 编译链路必须的新 runtime 导出符号。

若触发，执行两阶段：

1. 先发布含新能力但 `src/std` 未启用的编译器版本。
2. bump `bootstrap.lock` 并通过 rolling gate。
3. 再启用 `src/std` 调用并移除旧路径。

## 6. Atomic Deferred Backlog

- [ ] NIO-AT1 Atomic 从函数调用迁移为 IR/指令语义（Deferred）
  - [ ] 目标：`std/sync` 原子操作最终不依赖 `vox_host_atomic_*` 函数调用表面。
  - [ ] 前提：IR 指令语义与跨平台 lowering 方案稳定。
  - [ ] 风险：大概率触发编译器能力变更，需按发布两阶段执行。

## 7. 完成标准

每个 `NIO-*` 条目完成必须满足：

1. 实现：对应包职责与 API 边界满足本清单。
2. 测试：标准库行为测试 + typecheck/compile smoke 通过。
3. 文档：`13/16/17` 等相关文档同步。
4. CI：`make fmt`、`make test`、rolling gate 全绿。
5. 记录：在该文档勾选完成项并附关键落地文件。

## 8. 最新进展（2026-02-22）

本轮已落地：

- `std/net` 接管连接生命周期（`NetConn/Net*Result/TcpListener/SocketAddr.tcp_connect/listen/bind_udp`）。
- `std/io` 移除网络连接职责，仅保留 IO 基础 + `Reader/Writer/Closer/ReadWriter` 与 `BufReader/BufWriter` 基线。
- 已移除 `NetConn`/`File` 上冗余全局转发函数（优先方法风格调用，避免双入口 API）。
- `std/process` 的 `args/exe_path/getenv` 已切到 `std/os`。
- `std/os` 与 `std/time` 已去除 `vox_*` FFI，统一改为调用 `std/sys`。
- `std/sys` 的 `args/exe_path/getenv/read_file/walk_files/now_ns/yield_now/tcp_*` 已切到 `vox_impl_*` 薄桥。
- `std/time::yield_now` 已从 `c_runtime` 特殊实现下沉到各平台 `std/sys` 直接 FFI（linux/darwin/wasm: `sched_yield`，windows: `Sleep(0)`），并删除 `c_runtime` 中 `vox_impl_yield_now`。
- `std/runtime` 已去除 `args/exe/getenv/time/os/tcp/mutex` 入口，仅保留 intrinsic/wake/atomic。
- `std/sync::Mutex` 底层改为复用 atomic 句柄；`std/os` 已移除 `mutex_i32/i64_*`；`c_runtime` 已删除 `vox_impl/vox_host_mutex_i32/i64_*` 实现与导出。
- `c_runtime` 已删除 `vox_host_{args,exe_path,getenv,now_ns,yield_now,exec,walk_vox_files,read_file,tcp_*}` alias 导出（保留 wake/atomic host 兼容面）。
- `c_runtime` 已移除未再使用的 `vox_impl_exec`；`std/process` 语义继续通过 `sys.system`。
- `c_runtime` 已移除未被 std 调用链使用的 `vox_impl_write_file/vox_impl_path_exists/vox_impl_mkdir_p` 死代码段。
- `std/runtime` 的 wake/atomic FFI 已从 `vox_host_*` 切到 `vox_impl_*`，并删除 `c_runtime` 中对应 `vox_host_wake_*`、`vox_host_atomic_*` 纯转发别名导出。
- `vox_*` FFI gate 更新为仅允许 `std/runtime`、`std/sys/sys_common`。

关键落地文件：

- `src/std/net/net.vox`
- `src/std/net/net_test.vox`
- `src/std/io/io.vox`
- `src/std/io/io_test.vox`
- `src/std/process/process.vox`
- `src/std/runtime/runtime.vox`
- `src/std/os/os.vox`
- `src/std/sys/sys_common.vox`
- `scripts/ci/check-no-vox-ffi-outside-runtime.sh`

仍待完成（下一批）：

- `NIO-01`：`sys.accept` 与各平台 listen/accept 语义补齐。
- `NIO-03`：`io.copy/read_all/write_all`。
- `NIO-07`：`vox/typecheck` 与 `vox/compile` 的 std override smoke 同步到新分层。
