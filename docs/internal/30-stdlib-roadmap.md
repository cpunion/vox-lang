# Stdlib Roadmap (Execution Checklist)

Status: active.

本清单用于标准库持续演进，遵循 `docs/reference/style-guide.md`。

## S0 已落地基线

- [x] `std/io` 方法式 API（`File` / `NetAddr` / `NetConn`）+ free-function 兼容包装
- [x] `std/fs` 方法式 API（`Path`）+ free-function 兼容包装
- [x] `std/process` 方法式 API（`Command`）+ free-function 兼容包装
- [x] `std/fs` 虚拟文件系统基线（`FS`/`WritableFS` + `OsFS`/`MemFS` + 泛型 helper）
- [x] `std/net` 地址抽象基线（`NetProto`/`SocketAddr` + `tcp://`/`udp://` URI parse/render）
  - `UdpSocket.send_to/recv_from` 当前为占位语义，待后续 runtime 事件源接线

## S1 下一批（P0）

- [x] `std/net` 请求对象化：`Client`/`Request`/`Response` 方法式入口收敛
  - 保留 `http_get/http_roundtrip` 兼容包装
  - 增加解析/构造 API 的结构化错误入口（不破坏现有 panic 行为）
- [x] `std/io` 连接级安全释放语义增强
  - `NetConn.close()` 幂等并返回关闭后的 `NetConn`（`net_close` 同步）
  - 增加 checked API：`try_send/try_recv/try_wait_read/try_wait_write/try_close` 与公开结果类型
  - 补 `std/io` 行为测试 + `vox/typecheck` / `vox/compile` smoke 回归
- [x] `std/fs` 路径拼接与规范化 helper（避免调用方重复字符串拼接）
  - `Path.clean/join/base_name/dir_name/ext/stem/is_abs` + free-function 同名入口

## S2 中期（P1）

- [x] `std/process` 增加 `Command.cwd` / `Command.env_remove` / `Command.clear_env`
  - 支持命令级 cwd 前缀渲染（`cd <dir> && ...`）
  - 支持按键移除环境变量与清空环境变量集合
  - 补 `std/process` 行为测试 + `vox/typecheck` / `vox/compile` smoke 回归
- [x] `std/collections` 与 `std/string` 的 OOP 风格一致性收敛（保持兼容包装）
  - `StrView` / `Slice[T]` 方法式核心 API 已对齐 free-function
  - `Queue.contains`、`Set.add_all/contains_all` 已补齐并保留兼容包装
  - `Map.set/remove` 覆盖/删除路径已切换为原地更新以降低拷贝成本
- [ ] `std/testing` 增加模块化断言扩展（不破坏当前 `t.assert*`）

## 执行要求

每个条目完成标准：

1. 标准库实现完成（方法式 + 兼容包装策略一致）
2. `src/std/*` 行为测试覆盖
3. `src/vox/typecheck/*` 与 `src/vox/compile/*` smoke 覆盖
4. 文档同步更新
5. PR review + CI 绿后合并
