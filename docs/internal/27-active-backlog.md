# Stage2 Active Backlog (Canonical)

Status: active.

Purpose:
- This is the only active task list for compiler language/tooling evolution.
- Closed batches must not be re-listed here.

Do-not-relist batches:
- `docs/internal/archive/25-p0p1-closure.md` (items 1-12 closed)
- `docs/internal/archive/26-closure-1-4-7-9.md` (items 1-4/7-9 closed)

## Why Tasks Repeated Before

1. Multiple backlog files existed, but no single canonical active list.
2. “Spec draft/deferred” lines were mixed into implementation TODO scans.
3. “完成一批” lacked a stable ID system, so similar items reappeared with new wording.

Governance from now on:
1. Every active item has a stable ID (`Axx`).
2. Completion requires code + tests + docs + commit, then mark `[x]` here.
3. Deferred items stay in Deferred and are not counted as active scope.
4. If a closed item must reopen, add explicit reason + regression evidence.

Builtin end-state (agreed):
1. `vox_rt_*` runtime capability exits language builtin surface; standard-library implementation only.
2. Builtin/intrinsic keep scope is limited to compile-time reflection:
   - type relations
   - type predicates
   - target reflection
3. Atomic behavior is modeled as IR instruction semantics, not runtime builtin function surface.

## Active Scope (non-deferred)

### P0

- [x] A08 Async cancel/drop 语义细化（先完成可验证基线）
  - [x] A08-1 增加 cancel 路径顺序保证回归：`cancel_cleanup*` 必须先于 `cancel_return*` 生成。
  - [x] A08-2 增加 cleanup-only 路径回归：仅有 `cancel_cleanup` 时仍走默认可恢复返回，不回退 panic。
  - Source: `docs/internal/09-async-model.md`.

- [x] A09 宏展开回退路径收敛（减少静默 fallback）
  - [x] A09-1 `name!(...)` 已定位到目标模块但 callee 缺失时，macroexpand 直接报错（不再降级为 call sugar）。
  - [x] A09-2 增加对应回归测试与文档同步。
  - Source: `docs/internal/10-macro-system.md`.

- [x] A10 specialization 冲突诊断增强
  - [x] A10-1 impl overlap 报错补充 `rank_trace`，稳定展示“更特化/不可比较”关系。
  - [x] A10-2 typecheck/compile 回归覆盖 `rank_trace` 文案。
  - Source: `docs/internal/06-advanced-generics.md`.

- [x] A22 specialization where-comptime rhs 链式蕴含增强
  - [x] A22-1 头部比较支持 rhs 符号链传递推理（例如 `@size_of(T) <= @align_of(U)` 与 `@align_of(U) <= 16` 推导 `@size_of(T) <= 16`）。
  - [x] A22-2 `generics_test` 增加对应 overlap/specialization 回归，保证 strict-order 判定稳定。
  - Source: `docs/internal/06-advanced-generics.md`.

- [x] A23 verified const cast 编译期校验补齐
  - [x] A23-1 `const` 场景下 `as @verified(...)` 改为编译期执行谓词函数，不再统一拒绝。
  - [x] A23-2 谓词返回 `false` 时在 const-eval 阶段报错 `verified check failed`，并补 typecheck/compile 回归。
  - Source: `docs/internal/01-type-system.md`, `docs/reference/language/types/verified.md`.

- [x] A24 verified `char` 底层类型回归补齐
  - [x] A24-1 增加 `@verified(... ) char` 的 const/非 const 成功路径覆盖。
  - [x] A24-2 增加 `@verified(... ) char` 谓词失败路径（const-eval）覆盖。
  - Source: `docs/internal/01-type-system.md`, `docs/reference/language/types/verified.md`.

- [x] A25 `@range` 边界常量扩展到 `char` 字面量
  - [x] A25-1 parser 支持 `@range('a'..='z')` / `@range('\n'..='\r')` 形式，边界按 codepoint 归一化存储。
  - [x] A25-2 typecheck/codegen 回归覆盖 char-bound range（成功与越界失败路径）。
  - Source: `docs/internal/01-type-system.md`, `docs/internal/14-syntax-details.md`, `docs/reference/language/types/ranges.md`.

- [x] A11 泛型 pack 限制策略收敛
  - [x] A11-1 `type pack arity exceeds materialization limit` 文案统一到 `typecheck` 单一函数出口，移除重复拼接。
  - [x] A11-2 typecheck/const-eval/irgen 统一复用同一 limit+error 文案。
  - Source: `docs/internal/06-advanced-generics.md`.

- [x] A18 泛型 pack arity/shape 上限策略增强
  - [x] A18-1 对异构且需要 materialization 的 pack，限制判定从“显式总长度”升级为“有效 shape arity”（按实际投影/消费位置计算）。
  - [x] A18-2 typecheck/const-eval/irgen 三路径统一该判定策略，并补充长 pack + 投影场景回归。
  - Source: `docs/internal/06-advanced-generics.md`.

- [x] A12 `vox/internal/*` 下沉继续推进
  - [x] A12-1 CLI 主流程复用 `vox/internal/text.trim_space`，移除 `src/main.vox` 重复实现。
  - [x] A12-2 新增 `vox/internal/path`，并让 `src/main.vox` / `vox/macroexpand` 复用统一路径辅助逻辑（`base_name/dirname/join/slash-normalize/is_abs_path`）。
  - Source: `docs/internal/28-vox-libraries.md`.

- [x] A19 `vox/internal/text` 复用收敛（第二批）
  - [x] A19-1 新增 `txt.contains_str`，统一字符串集合包含判断 helper。
  - [x] A19-2 `main/compile/loader/typecheck/world/fmt/list/manifest` 复用 `txt.has_prefix/has_suffix/contains_str/trim_space`，减少重复实现与维护面。
  - Source: `docs/internal/28-vox-libraries.md`.

- [x] A21 `vox/internal/strset` 复用收敛（排序/去重）
  - [x] A21-1 新增 `strset.insert_sorted/sort/push_unique_sorted`，统一字符串集合排序与唯一化 helper。
  - [x] A21-2 `main` 与 `vox/list` 复用 `vox/internal/strset`，移除重复实现。
  - Source: `docs/internal/28-vox-libraries.md`.

- [x] A26 `vox/internal/text` 复用收敛（第四批）
  - [x] A26-1 `vox/typecheck/tc_struct_lit` 的字符串集合包含判断复用 `txt.contains_str`。
  - [x] A26-2 `vox/irgen/async_lower` 的字符串集合包含与前缀判断复用 `txt.contains_str/txt.has_prefix`。
  - Source: `docs/internal/28-vox-libraries.md`.

- [x] A27 `vox/internal/text` 复用收敛（第五批）
  - [x] A27-1 `vox/typecheck/world` 移除 `has_prefix/contains_str` 转发 helper，统一改为直接调用 `txt.has_prefix/txt.contains_str`。
  - Source: `docs/internal/28-vox-libraries.md`.

- [x] A28 `vox/internal/text` 复用收敛（第六批）
  - [x] A28-1 `vox/manifest` 移除 `has_prefix/has_suffix/contains_str` 转发 helper，统一改为直接调用 `txt.*`。
  - Source: `docs/internal/28-vox-libraries.md`.

- [x] A29 `vox/internal/text` 复用收敛（第七批）
  - [x] A29-1 `vox/manifest` 继续移除 `trim/index/split` 转发 helper，统一改为直接调用 `txt.trim_space/txt.index_byte/txt.split_*`。
  - Source: `docs/internal/28-vox-libraries.md`.

- [x] A30 `vox/internal/text` 复用收敛（第八批）
  - [x] A30-1 `main_toolchain/main_lock` 移除 `has_prefix/unquote/index/split` 转发 helper，统一改为直接调用 `txt.*`。
  - Source: `docs/internal/28-vox-libraries.md`.

- [x] A42 `vox/internal/text` 复用收敛（第九批）
  - [x] A42-1 `vox/loader` 与 `vox/compile` 移除 `has_prefix` 转发 helper，统一改为直接调用 `txt.has_prefix`。
  - Source: `docs/internal/28-vox-libraries.md`.

- [x] A13 规范与实现一致性修正
  - [x] A13-1 `docs/internal/09-async-model.md` 示例签名与当前 `EventRuntime` 默认行为一致。
  - [x] A13-2 `docs/internal/14-syntax-details.md` 成员调用说明更新为 trait/impl 已支持现状。
  - [x] A13-3 `docs/internal/10-macro-system.md` 补充 missing-callee 直错规则。
  - Source: `docs/internal/09-async-model.md`, `docs/internal/14-syntax-details.md`, `docs/internal/10-macro-system.md`.

- [x] A14 Async 事件驱动执行器（epoll/kqueue/IOCP + 多源就绪队列）
  - [x] A14-0 发布链路前置解锁（rolling bootstrap 两阶段）
    - [x] A14-0a 先发布包含 `__wake_notify/__wake_wait` 编译器支持的新版本（标准库暂不调用）。
      - Released: `v0.2.11` (2026-02-18).
    - [x] A14-0b 更新 `scripts/release/bootstrap.lock` 到该版本并通过 rolling 门禁。
    - [x] A14-0c 再放开 `src/std` 调用与 `bootstrap-intrinsics.allow`，确保第一跳不被旧 bootstrap 卡住。
      - Landed: `std/async` uses `__wake_notify/__wake_wait`, allowlist synced.
  - [x] A14-1 `EventRuntime` 接入 wake token 通路（`wake` -> `__wake_notify`，`park_until_wake` -> `__wake_wait`）。
    - Landed in `src/std/async/async.vox` + `src/std/async/async_test.vox`.
  - [x] A14-2 事件源抽象与多源就绪队列（为 epoll/kqueue/IOCP 统一接口做准备）。
    - Baseline landed: `EventSource` + `ReadyPoll` + `ReadyQueue` and queue bridge helpers in `std/async`.
  - [x] A14-3 平台实现补齐与回归（linux/macOS/windows + wasm 行为约束）。
    - Landed: wake runtime platform constraints documented in `docs/internal/16-platform-support.md` and locked by codegen regressions in `src/vox/codegen/c_emit_test.vox`.
  - [x] A14-4 真实事件源接线（C runtime）：
    - Linux `eventfd + epoll`、macOS/*BSD `kqueue(EVFILT_USER)`、Windows `IOCP`，并保留 wasm/其他平台回退分支；
    - `__wake_notify/__wake_wait` 保持 token pending 语义不变，仅替换底层等待机制。
  - Scope: 在保持现有 `Runtime` trait 兼容前提下，把 `EventRuntime` 从 timeout-yield 基线升级为真正事件驱动。
  - Source: `docs/internal/09-async-model.md`, `docs/internal/29-async-expr-await.md`.

- [x] A15 Async 事件源多 context 扫描接线基线
  - [x] A15-1 `std/async` 新增 `drain_ready_once`，将多个 `Context` 的单轮 `EventSource.wait` 结果统一入 `ReadyQueue`，减少宿主重复样板。
    - Landed in `src/std/async/async.vox` + `src/std/async/async_test.vox`.
  - Source: `docs/internal/09-async-model.md`.

- [x] A31 Async 事件源多 context 批量等待（`__wake_wait_any`）
  - [x] A31-1 C runtime 新增 `__wake_wait_any(tokens, timeout_ms) -> i32`，在 `epoll/kqueue/IOCP` 等待路径上复用单次平台等待 + token 扫描，返回命中下标。
  - [x] A31-2 typecheck/codegen/compile 回归覆盖新 intrinsic 与生成代码路径。
  - [x] A31-3 发布链路两阶段落地：先发布含 intrinsic 的编译器，再 bump bootstrap lock，最后放开 `src/std` 使用。
  - [x] A31-4 `src/std` phase-b 接线：`std/runtime` 提供 `wake_wait_any` 封装，`std/async::wait_many/drain_ready_once` 接入批量等待路径，并补回归测试。
  - Source: `docs/internal/09-async-model.md`.

- [x] A32 Socket 就绪等待 intrinsic（fd/socket 级事件接线基础）
  - [x] A32-1 compiler/runtime 侧新增 `__tcp_wait_read/__tcp_wait_write`，并在 C runtime 提供平台等待分支：
    - Linux: `epoll`（单 fd 一次等待）
    - macOS/*BSD: `kqueue`（`EVFILT_READ/EVFILT_WRITE` 一次等待）
    - Windows: `IOCP`（`CreateIoCompletionPort + WSARecv/WSASend(overlapped) + GetQueuedCompletionStatus`）
  - [x] A32-2 发布 + bootstrap lock bump 后，放开 `src/std/runtime` 与 `src/std/io` 的公开封装 API（`tcp_wait_read/tcp_wait_write` 与 `net_wait_read/net_wait_write`）。
  - Source: `docs/internal/09-async-model.md`, `docs/internal/16-platform-support.md`.

- [x] A16 Async cancel/drop 细化：frame 重绑定钩子
  - [x] A16-1 async entry/test wrapper 在取消分支新增可选 `cancel_drop_with/cancel_drop` 调用，并固定顺序为 `cancel_drop -> cancel_cleanup -> cancel_return`。
    - Landed in `src/vox/compile/compile.vox` with regressions in `src/vox/compile/async_test.vox`.
  - [x] A16-2 `std/async` 增加默认 no-op 的 `cancel_drop_with/cancel_drop`，并补标准库回归。
    - Landed in `src/std/async/async.vox` + `src/std/async/async_test.vox`.
  - Source: `docs/internal/09-async-model.md`.

- [x] A20 Async cancel/drop 细粒度资源回收策略（hint/state 分层）
  - [x] A20-1 `std/async::CancelHint` 增加 `reclaim` 字段，并提供 `cancel_reclaim_keep/shallow/deep` 与 `cancel_reclaim_from_state_spins`。
  - [x] A20-2 默认 hint 钩子下沉到 state 钩子：`cancel_drop_hint* -> cancel_drop_state*`、`cancel_cleanup_hint* -> cancel_cleanup_state*`，并补齐 `cancel_cleanup_{state_with,with,state}` 默认实现。
  - [x] A20-3 标准库回归覆盖 reclaim 计算与 cleanup state hooks，文档同步。
  - Source: `docs/internal/09-async-model.md`, `docs/reference/language/async-await.md`.

- [x] A17 Thread-safety marker negative impl（`!Send/!Sync`）
  - [x] A17-1 语法层支持 `impl !Trait for Type {}`（当前用于 `Send/Sync`）。
    - Landed in `src/vox/ast/ast.vox`, `src/vox/parse/parse.vox`, `src/vox/parse/parse_test.vox`.
  - [x] A17-2 语义层支持 negative impl 覆盖自动推导，并禁止 `Send/Sync` 的手写正向 impl。
    - Landed in `src/vox/typecheck/collect_traits_impls.vox`, `src/vox/typecheck/sym_lookup.vox`, `src/vox/typecheck/typecheck_test.vox`, `src/vox/compile/misc_test.vox`.
  - Source: `docs/internal/08-thread-safety.md`, `docs/internal/14-syntax-details.md`.

- [x] A01 Real generic pack expansion (type/value packs), not declaration-only.
  - [x] A01-1 Trailing explicit type args can bind a single trailing type pack.
    - Landed in `src/vox/typecheck/tc_call.vox`, `src/vox/irgen/gen_call_match.vox`, and `src/vox/typecheck/consts.vox`, with compile/typecheck tests covering both runtime and const-call paths.
  - [x] A01-2 Heterogeneous type pack binding + true per-position substitution model.
    - [x] A01-2a Allow heterogeneous trailing explicit type args when pack is only a placeholder (not materialized in params/ret/variadic/bounds).
    - [x] A01-2b True per-position substitution model for materialized heterogeneous packs.
      - [x] Runtime call, const-eval call, and IRGen all support per-position materialization for params/ret/variadic type slots.
      - [x] Heterogeneous pack instantiation names are disambiguated (`pack`, `pack__1`, ...), avoiding monomorph collisions.
      - [x] Pack projection members (`Pack.N`) in materialization are supported across parse/typecheck/compile paths.
      - [x] Heterogeneous pack participation in bounds/where clauses is supported (trait bounds + comptime where reflect).
  - [x] A01-3 Value pack expansion and call-site lowering coherence.
    - Verified by pack-call/vec-call dual-mode tests in `src/vox/typecheck/typecheck_test.vox` and `src/vox/compile/compile_test.vox`.
  - Source: `docs/internal/06-advanced-generics.md`.

- [x] A02 String/borrow model convergence from transitional `String/str` aliasing to compiler-stable borrow constraints and diagnostics.
  - [x] A02-1 Bare `str` is now rejected; use `String` for owned text and `&str`/`&'static str` for borrow-position text.
    - Covered in `src/vox/typecheck/ctx.vox`, with compile/typecheck regressions in `src/vox/typecheck/typecheck_test.vox` and `src/vox/compile/compile_test.vox`.
  - [x] A02-2 `&mut`/`&'static mut` call arguments now require mutable place roots (local mutable var or member-chain rooted at one), across direct calls, variadic paths, generic calls, and method-sugar dispatch.
    - Covered in `src/vox/typecheck/tc_call.vox`, with regressions in `src/vox/typecheck/typecheck_test.vox` and `src/vox/compile/compile_test.vox`.
  - [x] A02-3 Non-static `&T` call arguments now require place roots (identifier/member-chain rooted at local), across direct calls, generic calls, variadic paths, and method-sugar dispatch.
    - Covered in `src/vox/typecheck/tc_call.vox`, with regressions in `src/vox/typecheck/typecheck_test.vox` and `src/vox/compile/compile_test.vox`.
  - [x] A02-4 `let` annotations with non-static borrow now validate initializer sources (`&T` requires place; `&mut T` requires mutable place).
    - Covered in `src/vox/typecheck/tc_fn.vox`, with regressions in `src/vox/typecheck/typecheck_test.vox` and `src/vox/compile/compile_test.vox`.
  - [x] A02-5 Call-arg mismatch diagnostics are now borrow-aware: expected type text preserves borrow form (`&T`/`&mut T`/`&'static T`/`&'static mut T`) instead of showing erased base type.
    - Covered in `src/vox/typecheck/tc_call.vox`, `src/vox/typecheck/typecheck_test.vox`, and `src/vox/compile/compile_test.vox`.
  - [x] A02-6 closure note: borrow remains signature-metadata based in this stage; first-class borrow IR/type representation is deferred to `D06`.
  - Sources: `docs/internal/13-standard-library.md`, `docs/internal/20-bootstrap.md`, `docs/internal/19-ir-spec.md`.

- [x] A03 Runtime memory model convergence (compiler scope).
  - [x] A03-1 Runtime tracked allocations now support early release via `vox_rt_free`; non-escaping temp path buffers in `mkdir_p`/`walk_vox_files` are released eagerly instead of waiting for process exit.
    - Covered in `src/vox/codegen/c_runtime.vox` and `src/vox/codegen/c_emit_test.vox`.
  - [x] A03-2 `std/sync` handles now support explicit release (`mutex_drop`/`atomic_drop`) via new low-level drop intrinsics, reducing long-running tool memory retention without changing value semantics.
    - Covered in `src/vox/typecheck/collect.vox`, `src/vox/codegen/c_func.vox`, `src/vox/codegen/c_runtime.vox`, `src/std/sync/sync.vox`, `src/vox/codegen/c_emit_test.vox`, and `src/vox/smoke_test.vox`.
  - [x] A03-3 `vox_rt_free` now only frees tracked allocations (`vox_rt_forget` returns bool), so duplicate release on copied sync handles becomes idempotent instead of double-free.
    - Covered in `src/vox/codegen/c_runtime.vox`, `src/vox/codegen/c_emit_test.vox`, and `src/vox/smoke_test.vox`.
  - [x] A03-4 `std/sync` handles now use runtime liveness registry (`vox_sync_handle_add/live/remove`): sync ops panic on dropped/invalid handles, and drop is remove-gated for deterministic idempotence.
    - Covered in `src/vox/codegen/c_runtime.vox` and `src/vox/codegen/c_emit_test.vox`.
  - [x] A03-5 Sync-handle registry nodes now use tracked runtime allocation (`vox_rt_malloc/vox_rt_free`), so undisposed-handle paths do not leave untracked registry memory behind.
    - Covered in `src/vox/codegen/c_runtime.vox` and `src/vox/codegen/c_emit_test.vox`.
  - [x] A03-6 closure note: full ownership/move/drop for general values/containers is deferred to `D07` to keep rolling-bootstrap stable.
  - Source: `docs/internal/20-bootstrap.md`.

- [x] A04 Package registry remoteization.
  - [x] A04-1 Registry dependencies now support remote git-backed registry roots (`git+...`/URL/`.git`) with clone/fetch cache under `.vox/deps/registry_remote`, then resolve `name/version` from cached checkout.
    - Covered in `src/main.vox`; historical selfhost integration evidence is archived.
  - Source: `docs/internal/11-package-management.md`.

### P1

- [x] A05 Macro system closure from MVP to stable full execution model (while keeping deterministic diagnostics).
  - [x] A05-1 Expression-site macro execution is now strictly typed: macro fns returning `AstStmt/AstItem` are rejected at expression macro call sites with deterministic diagnostics (`macro call requires AstExpr or AstBlock return type; got ...`).
    - Covered in `src/vox/macroexpand/macroexpand.vox`, `src/vox/macroexpand/user_macro_inline.vox`, and tests in `src/vox/macroexpand/macroexpand_test.vox`.
  - [x] A05-2 Statement-site `name!(...)`/`compile!(...)` now accepts `AstStmt` return type (direct `ExprStmt` positions), while expression sites remain `AstExpr/AstBlock`-only.
  - Source: `docs/internal/10-macro-system.md`.

- [x] A06 Diagnostics span coverage completion (remaining weak paths in typecheck/irgen).
  - [x] A06-1 Call-site diagnostics now emit concrete reasons for argument/type-arg failures instead of generic `typecheck failed` in common paths.
    - Covered in `src/vox/typecheck/tc_call.vox`, `src/vox/typecheck/typecheck_test.vox`, `src/vox/compile/compile_test.vox`.
  - [x] A06-2 Reserved intrinsic/private prelude function call paths now report explicit type errors.
  - [x] A06-3 Member/struct-literal diagnostics upgraded from generic fallback to explicit unknown/private/type-mismatch messages.
    - Covered in `src/vox/typecheck/tc_member.vox`, `src/vox/typecheck/tc_struct_lit.vox`, `src/vox/typecheck/tc_expr.vox` with paired tests in typecheck/compile suites.
  - [x] A06-4 Enum constructor diagnostics (`.Variant(...)` and `Enum.Variant(...)`) now emit explicit unknown-variant/arity/arg-mismatch/result-mismatch errors.
    - Covered in `src/vox/typecheck/tc_call.vox` with paired tests in `src/vox/typecheck/typecheck_test.vox` and `src/vox/compile/compile_test.vox`.
  - Source: `docs/internal/18-diagnostics.md`.

- [x] A07 Specialization rule strengthening (where-strength/ordering edge cases).
  - [x] A07-1 Reject impl head type params that are unconstrained by `for` target type; this removes ambiguous overlap that can be introduced only via extra impl-head params/bounds.
    - Covered in `src/vox/typecheck/collect_traits_impls.vox` with paired tests in `src/vox/typecheck/generics_test.vox` and `src/vox/compile/compile_test.vox`.
  - Source: `docs/internal/06-advanced-generics.md`.

- [x] A33 `@deprecated` 扩展到方法（trait/impl/inherent）并补齐调用点告警
  - [x] A33-1 parser 放开方法上的 `@deprecated`（保留参数规则：`@deprecated` / `@deprecated("msg")`）。
  - [x] A33-2 告警从“仅 AST 名字扫描”补强到“按已解析调用目标”路径（IR call target），覆盖 `x.m()`/`Type.m(x)`/trait default/async lowered method。
  - [x] A33-3 回归测试补齐：parser（trait/impl/inherent 方法）与 compile（方法调用告警），并同步文档。
  - Source: `docs/internal/14-syntax-details.md`, `docs/internal/18-diagnostics.md`.

- [x] A34 `@deprecated` 告警可读性与覆盖补充
  - [x] A34-1 告警文案补充调用者函数名（`called from`），提升 `<unknown>` 位置信息场景下可读性。
  - [x] A34-2 compile 回归补充：顶层函数调用与 trait default 方法调用的 `@deprecated` 告警覆盖。
  - [x] A34-3 diagnostics 文档补充 warning 格式与 `W_DEPRECATED_0001` 稳定码说明。
  - Source: `docs/internal/18-diagnostics.md`.

- [x] A35 builtin/intrinsic 冻结门禁
  - [x] A35-1 新增 `scripts/release/frozen-builtins.lock`，固化 `collect.vox` 当前 builtin/intrinsic 符号集合。
  - [x] A35-2 新增 `scripts/ci/check-frozen-builtins.sh`，若集合发生增删则 CI 失败。
  - [x] A35-3 `Makefile` 与 CI workflow 接入冻结检查，防止未审阅的 builtin/intrinsic 扩张进入主线。
  - Source: `docs/reference/style-guide.md`.

- [x] A36 builtin/intrinsic 收缩（batch-1）
  - [x] A36-1 移除 `__intrinsic_abi` / `__has_intrinsic` 两个 runtime-backed builtin 声明与 codegen 分支。
  - [x] A36-2 同步删减 C runtime 里对应 `vox_builtin_*` 实现，保留 `std/runtime` 语言层能力。
  - [x] A36-3 更新冻结清单与回归测试（`typecheck`/`codegen`）。

- [x] A37 builtin/intrinsic 收缩（batch-2）
  - [x] A37-1 将 `__exe_path/__getenv/__now_ns/__yield_now/__exec` 从 builtin 路径迁到 `std/runtime` 的 `@ffi_import("c", "vox_builtin_*")` 调用。
  - [x] A37-2 删除上述符号在 `collect` 与 `c_func` 的 builtin 声明/硬编码 lowering，改走常规 FFI 调用路径。
  - [x] A37-3 `c_runtime` 将对应 `vox_builtin_*` 实现改为可外部链接，满足编译器自举二进制内 FFI 绑定。
  - [x] A37-4 更新冻结清单并补齐回归（`typecheck/compile/codegen` 的 std override 与 yield 场景）。
  - Source: `docs/reference/style-guide.md`.

- [x] A38 builtin/intrinsic 收缩（batch-3）
  - [x] A38-1 将 `__read_file/__write_file/__path_exists/__mkdir_p` 从 builtin 路径迁到 `std/runtime` 的 `@ffi_import("c", "vox_builtin_*")` 调用。
  - [x] A38-2 删除上述符号在 `collect` 与 `c_func` 的 builtin 声明/硬编码 lowering，改走常规 FFI 调用路径。
  - [x] A38-3 `c_runtime` 将对应 `vox_builtin_*` 实现改为可外部链接，满足编译器自举二进制内 FFI 绑定。
  - [x] A38-4 更新冻结清单并补齐回归（`typecheck/compile` 的 std io/fs override 场景）。
  - Source: `docs/reference/style-guide.md`.

- [x] A39 builtin/intrinsic 收缩（batch-4）
  - [x] A39-1 扩展 FFI 类型白名单，支持 `Vec[String]` 的参数与返回（用于 runtime bridge，不放开任意 `Vec[T]`）。
  - [x] A39-2 在 bootstrap lock 升级后，将 `__args/__walk_vox_files` 从 builtin 路径迁到 `std/runtime` 的 `@ffi_import("c", "vox_builtin_*")` 调用。
  - [x] A39-3 同步删除上述符号在 `collect` 与 `c_func` 的 builtin 声明/硬编码 lowering，并更新冻结清单。
  - Source: `docs/reference/style-guide.md`.

- [x] A40 builtin/intrinsic 收缩（batch-5）
  - [x] A40-1 将 `std/runtime` 的 `sync` 句柄族（`mutex/atomic` i32+i64）改为 `@ffi_import("c", "vox_builtin_*")` 调用。
  - [x] A40-2 删除上述符号在 `collect` 与 `c_func` 的 builtin 声明/硬编码 lowering，改走常规 FFI 路径。
  - [x] A40-3 `c_runtime` 对应 `vox_builtin_*` 实现改为可外部链接（非 `static`），满足同一产物内 FFI 绑定。
  - [x] A40-4 更新冻结清单并补回归（`typecheck/compile/codegen` 同步场景）。
  - Source: `docs/reference/style-guide.md`.

- [x] A41 builtin/intrinsic 收缩（batch-6）
  - [x] A41-1 将 `std/runtime` 的 `tcp` 与 `wake_notify/wake_wait` 路径改为 `@ffi_import("c", "vox_builtin_*")` 调用。
  - [x] A41-2 删除上述符号在 `collect` 与 `c_func` 的 builtin 声明/硬编码 lowering。
  - [x] A41-3 `c_runtime` 对应 `vox_builtin_*` 实现改为可外部链接（非 `static`），并补齐 `typecheck/compile/codegen` 回归。
  - [x] A41-4 更新冻结清单（移除 `__wake_wait_any`；仅保留 `panic/print` 与反射内建）。
  - Source: `docs/reference/style-guide.md`.

- [x] A43 builtin/intrinsic 收缩（batch-7，bootstrap 兼容前置）
  - [x] A43-1 C runtime 增加 `vox_rt_print` 外部桥接符号（内部复用 `vox_builtin_print` 实现），为后续 `print` 去 builtin 化做 bootstrap 前置准备。
  - [x] A43-2 说明：`print` 语言 builtin 的移除与调用路径切换需要“先发布再 bump bootstrap.lock”的两阶段落地，避免锁定 bootstrap 链路断裂。
  - [x] A43-3 发布桥接版本 `v0.2.19`（包含 `vox_rt_print`）并验证多平台 release 产物。
  - [x] A43-4 更新 `scripts/release/bootstrap.lock` 到 `v0.2.19`，使后续 `print` 去 builtin 化可在锁定 bootstrap 下安全落地。
  - [x] A43-5 移除 `print` builtin（`collect/c_func/frozen lock`），由 `std/prelude::print` 通过 `@ffi_import("c", "vox_rt_print")` 提供并补齐回归测试。
  - Source: `docs/reference/style-guide.md`.

- [ ] A44 字符串 FFI/运行时收敛：从 C-string 语义迁移到 `ptr + len`，并优先平台 API
  - [ ] A44-1 约束收敛：标准库/FFI 设计不再依赖“字符串必须 `\\0` 结尾”；跨边界文本/字节接口统一采用 `(ptr,len)` 视图（必要时由适配层显式构造终止符缓冲）。
  - [x] A44-1a `std/sys` 新增跨平台 `write(fd, const rawptr, len)`，`std/prelude::print` 改为 `String -> const rawptr + len` 调用，不再依赖 `vox_builtin_print` 绑定。
  - [x] A44-1b `std/net::NetConn.try_send` 已改为 `std/sys::socket_send(handle, text as const rawptr, len)`，平台分支直接走 OS `send`（`@build`），避免向 `c_runtime` 扩增网络桥接符号。
    - 注：windows-x86 仍为占位分支（winsock `send` 在 x86 为 stdcall，当前 FFI 尚不支持按符号 calling convention）。
  - [x] A44-1c `std/runtime` 收敛：移除未被 `std/*` 使用的 legacy facade（`tcp_send/write_file/path_exists/mkdir_p`）并同步收缩 `c_runtime` 对应 alias 导出。
  - [x] A44-1d `std/sys::socket_send` 签名收敛为 `socket_send(handle, const rawptr, len)`；`std/net::NetConn.try_send` 在调用点显式完成 `String -> const rawptr + len` 适配，缩减 `String` 形参跨层传递。
  - [ ] A44-2 平台抽象收敛：`std/sys` 以平台原生 API 为主（Linux syscall/Unix/POSIX、Darwin、Windows API），避免将 `libc` 作为默认统一抽象层。
  - [x] A44-2a `std/fs::write_string` 改为 `std/sys::creat + write + close` 路径（linux/darwin/windows/wasm 平台分流），`exists/mkdir_p` 已在前序改造走 `std/sys::access/mkdir`。
  - [x] A44-2b `std/io::File` 文件能力（`exists/read_all/write_all/mkdir_p`）统一委托给 `std/fs`，移除对 `std/runtime` 文件接口的直接调用。
  - [x] A44-2c `std/sys` 增加 `open_read(path)` 接口：linux/darwin/wasm 已接通 `open(O_RDONLY)`；Windows 分架构实现已接通（amd64 `_sopen_dispatch`，x86 `_sopen_s(&mut i32, ...)`）。
    - Landed: `src/std/sys/sys_windows.vox`, `src/std/sys/sys_windows_x86.vox`, `src/std/sys/sys_test.vox`.
  - [x] A44-3 编译器与标准库边界收敛：新增/迁移后不引入新的 `vox_builtin_*` / `vox_rt_*` 功能面；同等能力优先通过 `@ffi_import + @build` 在 `std/*` 实现（`@cfg` 仅保留测试）。
    - Landed: runtime-alias gate + builtin-alias gate (`scripts/ci/check-no-vox-rt-in-src.sh`, `scripts/ci/check-no-vox-builtin-in-src.sh`) wired in `make test`.
    - Landed: `src/vox/codegen/c_runtime.vox` 移除未被源码使用的 `vox_builtin_* -> vox_host_*` 兼容别名导出。
    - Landed: `src/vox/codegen/c_runtime.vox` 移除未被源码引用的 `vox_impl_print` 导出与内部 `vox_host_print` helper。
    - Landed: `src/vox/codegen/c_runtime.vox` 移除未被源码使用的 `vox_host_sys_write` 导出。
  - [x] A44-4 回归与文档：补齐 `std/sys` + `std/fs` + FFI 相关测试，文档明确“何时需要 NUL 适配缓冲、何时直接 `ptr+len`”。
    - Landed: `src/std/sys/sys_test.vox::test_sys_write_ptr_len_controls_written_bytes` + docs sync in `docs/internal/17-ffi-interop.md` and `docs/reference/language/attributes-ffi.md`.
  - [ ] A44-5 最终目标：移除对 `libc` 的运行时依赖（含默认 `libc` I/O/socket/path 兜底路径）；平台实现以系统调用/原生 OS API 为主，仅在无法避免处保留最小兼容垫片并单独标注。
  - Source: `docs/internal/17-ffi-interop.md`, `docs/internal/13-standard-library.md`, `docs/internal/16-platform-support.md`.

- [x] A45 生成 C 代码低效点基线分析 + 按包编译/编译缓存规划
  - [x] A45-1 新增设计文档：记录当前 C 产物体量、冗余模式与优化优先级（cast/bitcast/wrapper/CFG）。
  - [x] A45-2 明确按包增量构建三层缓存设计：`pkg-sem`、`pkg-obj`、`link`，并给出 key/invalidation 规则。
  - [x] A45-3 给出分阶段落地顺序与门禁要求，作为后续实现 PR 的单一基线。
  - Source: `docs/internal/31-codegen-c-efficiency-and-package-cache-plan.md`.

## Deferred Scope

- [x] D01 `--target` CLI, target triples, linker config, cross-compilation matrix.
  - Landed: build/test/run/install accept `--target`; parser supports GNU/MinGW + MSVC Windows triples; codegen chooses toolchain/flags/link options by target and compiler family.
  - Source: `docs/internal/16-platform-support.md`.

- [x] D02 Thread-safety model (`Send/Sync` auto-derivation policy).
  - Stage2 baseline landed: marker traits in `std/prelude` + auto-derivation for scalars/String/Vec/Range/struct/enum; type params still require explicit bounds; `impl !Send/!Sync` supported and explicit positive `impl Send/Sync` rejected.
  - Source: `docs/internal/08-thread-safety.md`.

- [x] D03 Async model.
  - [x] D03-1 词法/语法前置：保留 `async`/`await` 关键字，`async fn` AST 标记接入。
  - [x] D03-2 未启用语义的稳定诊断：`await`/trait async method 给出明确 deferred 错误。
  - [x] D03-2a `await` 语法通路：parser 产出 `ExprNode.Await`（表面语法推荐 `e.await`，保留 `await e` 兼容），由 typecheck/irgen 统一给 deferred 语义错误。
  - [x] D03-3b0 async fn 管线打通（scaffold）：`async fn`（无 `await`）进入正常 typecheck/codegen 流程；完整 frame/poll lowering 仍在 D03-3b。
  - [x] D03-3c0 await 脚手架接入：`await` 仅允许出现在 `async fn`；在当前阶段按同步直通表达式进入 typecheck/irgen（真实 Future 语义仍在 D03-3b/3c）。
  - [x] D03-3 Future 表示与 lowering（状态机/poll 模型）。
  - [x] D03-3a `std/async` pull-core 契约落地：`Poll[T]`、`Waker`、`Context`、`Future`、`Sink` 与基础 helper。
  - [x] D03-3b `async fn` lowering 到状态机 frame + `poll`。
  - [x] D03-3c `await` 的 typecheck/irgen 接入：仅允许 `async fn`；操作数支持 Poll-shaped 枚举 `{ Pending, Ready(T) }`，或实现 `std/async::Future` 的类型；`Ready(v)` 提取为 `T`；`Pending` 从 enclosing `poll` 返回 `Pending` 并保留进度。
  - [x] D03-3c1 从 Poll scaffold 过渡到 Future trait（`Future::Output`）绑定。
  - [x] D03-4 借用跨 `await` 约束与诊断。
  - [x] D03-5 `await` in general expression control-flow (`try`/`match`/macro args).
    - Landed: async normalization now supports `await` in nested `if`/`match`/`try` expression control-flow and macro-call args, with compile/typecheck regressions in `src/vox/compile/async_test.vox` and `src/vox/typecheck/async_test.vox`.
    - See: `docs/internal/29-async-expr-await.md`.
  - Source: `docs/internal/09-async-model.md`.

- [x] D04 Effect/resource system.
  - [x] D04-1 Effect baseline landed:
    - `@effect(...)` on top-level functions, trait methods, and impl methods.
    - call-site effect checking in typecheck (`missing effect(s)` diagnostics).
    - trait impl methods must match trait method effect set.
  - [x] D04-2 Resource baseline landed:
    - `@resource(read|write, Name)` on top-level functions, trait methods, and impl methods.
    - call-site resource checking in typecheck (`resource check failed` diagnostics).
    - trait impl methods must match trait method resource read/write sets.
  - [x] D04-3 Resource/effect advanced model pending:
    - [x] D04-3a Declarative graph integration baseline:
      - `vox/list` module graph now exports module-level capability summaries (`effects`, `resource_reads`, `resource_writes`) for tooling and scheduling analysis.
    - [x] D04-3b resource scheduling/ordering constraints.
      - [x] `vox/list` 增加模块级资源冲突分析输出（`rw`/`ww`，`resource_conflicts`），作为并行调度前置输入。
      - [x] `vox/list` 增加函数级能力与冲突输出（`functions` / `function_resource_conflicts`），用于更细粒度调度分析。
      - [x] `vox/list` 增加模块级资源顺序建议输出（`resource_orders`，方向为 `dep -> importer`），用于在已知依赖边下给出冲突资源的保守串行顺序。
    - [x] D04-3c effect classes/executor integration (IO/GPU/etc.) and optimization semantics.
      - [x] `vox/list` 基线 effect class 分类输出：模块级/函数级新增 `effect_classes`（当前按 effect 命名约定映射 `IO/GPU/Async/Other`）。
      - [x] `vox/list` 基线执行器 lane 建议输出：`executor_lanes`（`class/executor/modules`），用于外部调度器按 effect class 分流。
      - [x] `vox/list` 基线模块调度提示输出：`module_schedule_hints`（`parallel_ok/serial_guarded`），结合资源冲突给出保守串并行建议。
  - Source: `docs/internal/00-overview.md`.

- [x] D05 FFI/WASM detailed ABI/attribute model.
  - [x] D05-1 统一属性签名与约束（位置/组合/泛型/符号唯一性）。
  - [x] D05-2 ABI 白名单与类型映射明确化（C 后端基线）。
  - [x] D05-3 wasm import/export 代码生成别名规则与 target 运行约束文档化。
  - Source: `docs/internal/17-ffi-interop.md`.

- [x] D06 First-class borrow IR/type representation (`&T`/`&str` non-erased types, borrow-aware IR ops).
  - [x] D06-1 Type-pool level borrow representation landed: `ir::TyKind.Ref` + `resolve_type` preserves `&T/&mut T/&'static T/&'static mut T` and reflection (`@type_name/@type`) can observe borrow shape.
  - [x] D06-2 Stage2 bootstrap boundary updated: irgen now preserves `Ref` in IR signatures/slots/calls, while `Range` continues to lower to base + `range_check` for v0 stability.
  - [x] D06-3 Borrow-aware IR/codegen landed: codegen `Ref` transparent type mapping + compare/nominal-eq borrow-aware unwrapping, with regression tests for IR signature preservation and `&str` compare codegen.
  - Extracted from A02 closure note.
  - Source: `docs/internal/19-ir-spec.md`, `docs/internal/13-standard-library.md`.

- [x] D07 Full ownership/move/drop semantics for general values/containers.
  - [x] D07-1 Remove bootstrap-safe `std/collections/map` fallback: switch to direct `Vec.set/remove/clear` implementation and keep `bootstrap -> compiler` gate green.
  - [x] D07-2 Container-level deterministic release model (Vec/String/Map) that is alias-safe under current value-copy semantics.
    - [x] D07-2a Deep-clone baseline landed: `Clone` trait + `impl[T: Clone] Clone for Vec[T]` + `impl[K: Eq + Clone, V: Clone] Clone for Map[K,V]` for explicit non-aliasing copy paths.
    - [x] D07-2b Add deterministic release semantics compatible with the current value-copy model (no UAF on alias copies).
      - Landed baseline: prelude `Release` trait + `release(String)`/`release_vec(Vec[T])`/`Map.release()` reset APIs; release is alias-safe and idempotent, while physical shared-storage reclaim remains deferred to D07-3 ownership/move/drop model.
  - [x] D07-3 Language-level ownership/move/drop rules and diagnostics (no-UAF contract) across function boundaries and aggregates.
    - [x] D07-3a Release API rebind enforcement: expression-statement `release` calls are rejected (`release call result must be assigned back`) to avoid silent non-rebinding misuse.
    - [x] D07-3b Minimal move-after-release diagnostics: values consumed by `release` are marked moved; later reads error as `use of moved value: <name>`, while `x = release(x)` remains a valid self-rebind path.
    - [x] D07-3c Move-state propagation baseline in control flow: `block/if/while` now conservatively propagate moved flags for outer locals, so branch/loop release paths are visible to later reads.
    - [x] D07-3d Aggregate-root move propagation for release paths: `release(x.field)` / `x.field.release()` now mark root `x` as moved (conservative no-UAF baseline under current copy semantics).
    - [x] D07-3e Member-chain moved-value diagnostics propagation: receiver member/call paths now preserve upstream `use of moved value` diagnostics instead of degrading to enum/path fallback errors.
    - [x] D07-3f Expression-shape move-source detection: `if`/block/match and nested call-expression trees now participate in release source discovery, so `let _ = if ... { release(x) } ...` and block-expression release paths mark `x` moved.
    - [x] D07-3g UFCS release parity: `Release.release(x)` now participates in move-source detection with the same moved-value diagnostics/self-rebind behavior as `release(x)` and `x.release()`.
    - [x] D07-3h Multi-source release tracking: expressions containing multiple release paths now mark all consumed roots (including call-arg fanout and `assign field` RHS), so moved diagnostics are not silently dropped for later sources.
  - Extracted from A03 closure note.
  - Source: `docs/internal/20-bootstrap.md`.

## Merged Backlog Sources (2026-02-23)

To keep a single backlog file in repository, the following backlog files were merged into this document on 2026-02-23:

- `docs/internal/31-net-io-os-runtime-backlog.md` (active domain backlog)
- `docs/internal/archive/22-backlog.md` (archived)
- `docs/internal/archive/23-backlog-next.md` (archived)

### Source Snapshot: `docs/internal/31-net-io-os-runtime-backlog.md`

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
  - [x] `send` 已先收敛为 `socket_send(handle, ptr, len) -> isize`（全平台分支同步）。
  - [x] `connect/recv/close_socket/wait_read/wait_write` 已收敛到 `std/sys` 入口，`std/net` 不再直接绑定 `vox_impl_tcp_*`。
  - [ ] linux/darwin/windows/wasm 分支补齐，无法支持项明确 panic/stub 语义。
  - [ ] `std/sys` 测试补齐接口 smoke 与失败语义。

- [x] NIO-02 `std/net` 承载连接生命周期
  - [x] `NetConn`/连接方法迁入并在 `net` 作为主入口。
  - [x] `SocketAddr.listen` 接入 `sys.*`；`tcp_connect` 当前在 `net` 内本地 FFI（后续随 NIO-01 延伸继续收敛）。
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
  - [x] `vox/typecheck` 与 `vox/compile` smoke 同步。
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

## 8. 最新进展（2026-02-23）

本轮已落地：

- `std/net` 接管连接生命周期（`NetConn/Net*Result/TcpListener/SocketAddr.tcp_connect/listen/bind_udp`）。
- `std/io` 移除网络连接职责，仅保留 IO 基础 + `Reader/Writer/Closer/ReadWriter` 与 `BufReader/BufWriter` 基线。
- 已移除 `NetConn`/`File` 上冗余全局转发函数（优先方法风格调用，避免双入口 API）。
- `std/process` 的 `args/exe_path/getenv` 已切到 `std/os`。
- `std/sys` 已去除 `args/exe_path/getenv/read_file/walk_files/now_ns` 高层桥接入口，仅保留薄层 syscall/API（含 `system/calloc/free/open_read/read/write/close/...`）。
- `std/sys` 网络发送入口统一为 `socket_send(handle, const rawptr, len) -> isize`（linux/darwin/windows/wasm/x86 分支已同步）。
- `std/sys` 已补齐 `connect/recv/close_socket/wait_read/wait_write` 网络薄入口（平台分支实现/占位语义）。
- `std/net` 当前 `connect/send/recv/close/wait_*` 全部经 `std/sys` 路径，不再在 `std/net` 直接绑定 `vox_impl_tcp_*`。
- `std/os` 当前仅承载 `args/exe_path/getenv`（通过 `vox_impl_*` FFI）；文件语义 `read_file/walk_files` 已下沉到 `std/fs` 内部；`std/time::now_ns` 当前通过 `vox_impl_now_ns`。
- `std/time::yield_now` 已从 `c_runtime` 特殊实现下沉到各平台 `std/sys` 直接 FFI（linux/darwin/wasm: `sched_yield`，windows: `usleep(0)`），并删除 `c_runtime` 中 `vox_impl_yield_now`。
- `std/runtime` 已去除 `args/exe/getenv/time/os/tcp/mutex` 入口，仅保留 intrinsic/wake/atomic。
- `std/sync::Mutex` 底层改为复用 atomic 句柄；`std/os` 已移除 `mutex_i32/i64_*`；`c_runtime` 已删除 `vox_impl/vox_host_mutex_i32/i64_*` 实现与导出。
- `c_runtime` 已删除 `vox_host_{args,exe_path,getenv,now_ns,yield_now,exec,walk_vox_files,read_file,tcp_*}` alias 导出（保留 wake/atomic host 兼容面）。
- `c_runtime` 已移除未再使用的 `vox_impl_exec`；`std/process` 语义继续通过 `sys.system`。
- `c_runtime` 已移除未被 std 调用链使用的 `vox_impl_write_file/vox_impl_path_exists/vox_impl_mkdir_p` 死代码段。
- `std/runtime` 的 wake/atomic FFI 已从 `vox_host_*` 切到 `vox_impl_*`，并删除 `c_runtime` 中对应 `vox_host_wake_*`、`vox_host_atomic_*` 纯转发别名导出。
- `c_runtime` 已删除未再使用的 `vox_impl_wake_wait_any` 及其扫描辅助函数，`std/runtime` 侧统一使用 `wake_wait` 组合实现 `wake_wait_any`。
- `vox_*` FFI gate 已按当前分层更新为允许 `std/runtime`、`std/fs/file_common`、`std/os`、`std/time` 与 `std/sys` 平台桥接（其余路径禁止）。
- `vox/compile` 与 `vox/typecheck` 的 std override smoke 已去除遗留 `vox_host_*`（`tcp_send/path_exists/write_file/mkdir_p`），统一改为直接 `c` FFI（`send/access/creat/write/close/mkdir`）或薄封装。
- `vox/compile` 与 `vox/typecheck` 的非核心 smoke 已进一步去除非必要 `vox_impl_*` 绑定（`read_file/walk_vox_files/args/exe_path/getenv/tcp_connect`），改为本地 stub，保留仅用于 runtime 能力覆盖的 `vox_impl_*` 用例。
- `vox/compile/std_smoke_override_test.vox`、`vox/typecheck/typecheck_test.vox`、`vox/typecheck/ffi_attr_test.vox`、`vox/compile/module_visibility_test.vox` 已清空 `vox_impl_*` 绑定；当前剩余 `vox_impl_*` 主要在 `std/runtime/std/os/std/time/std/fs` 与 `std/sys` 平台桥接，以及 runtime codegen 专项测试（`c_emit`）。

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
- `NIO-04`（延伸）：`walk_vox_files` 语义继续从标准库边界收敛到编译器内部实现（`vox/internal/*`），标准库仅保留通用文件遍历语义。
- `NIO-03`：`io.copy/read_all/write_all`。
- `NIO-LANG-01`：`time` 数值后缀糖（`3.seconds`）目前在 typecheck 报 `invalid member access`，需编译器新增“数值字面量成员单位糖”支持后才能启用。
- `NIO-LANG-02`：支持 Go 风格 `3 * time.s -> time.Duration`（不依赖操作符重载）：单位常量为 `Duration`，并补齐常量/字面量到 `Duration` 的二元算术类型规则。
- `NIO-LANG-03`：操作符重载能力（如 trait/协议驱动的 `+ - * / ==`）单独立项；当前 `time.Duration` 方案不依赖该能力。

### Source Snapshot: `docs/internal/archive/22-backlog.md`

# Stage2 Backlog (1-12)

Status: **archived (closed)**.
Canonical closure + gate: `docs/internal/archive/25-p0p1-closure.md`, `make test-p0p1`.

This file was the active burn-down list for compiler.  
Rule: complete one item end-to-end (code + tests + commit), then move to the next.

## Items

1. [x] Parser trailing comma completeness for generic call args (`f[T,](...)`, `f[3,](...)`, `m[T,]!(...)`).
2. [x] Macroexpand diagnostics: surface inline-fallback reason (why template inline was rejected).
3. [x] Macro execution v1: support function-like macro bodies returning expandable AST values (without `macro` keyword).
4. [x] `quote` / unquote MVP: expression-level quote with `$x` interpolation.
5. [x] Comptime execution expansion: broaden compile-time evaluable function shapes (pure subset).  
   Done scope: const/comptime evaluator now executes pure member-call subset (`String.len/byte_at/slice/concat/escape_c/to_string`, primitive `to_string`) inside const fn paths.
6. [x] Generic specialization diagnostics: deterministic conflict/ambiguity reports and ranking traces.  
   Done scope: impl candidate text now stable-sorted; ambiguity diagnostics include `rank_trace` with pairwise specificity relation.
7. [x] Generic packs/variadics design MVP (parser + typecheck skeleton, no codegen specialization yet).  
   Done scope: parser now accepts `T...` type-parameter packs and `arg: T...` variadic params; typecheck emits explicit skeleton diagnostics (no IR/codegen yet).
8. [x] Diagnostics upgrade: rune-aware column mapping and tighter span for type/const/macro errors.  
   Done scope: const block stmt executor now reports stmt-anchored spans (`let/assign/assign field/if/while/break/continue`), and macroexpand max-round overflow now reports first macro callsite span instead of fallback `1:1`.
9. [x] Testing framework upgrade: richer `--json` payload and stable rerun metadata pipeline.  
   Done scope: `test-pkg --json` now emits `report_version` + rerun-cache metadata fields, failed result entries include `error/log_file`, and rerun cache is versioned JSON (`version/updated_unix_us/tests`) with backward-compatible read + normalized load.
10. [x] Stdlib `std/sync`: generic `Mutex[T]` / `Atomic[T]` runtime-backed semantics on compiler.  
    Done scope: `std/sync` provides `Mutex[T: SyncScalar]` / `Atomic[T: SyncScalar]` generic handles (with `i32/i64` impls), runtime-backed intrinsics for load/store/fetch_add/swap, plus concrete compatibility wrappers.
11. [x] Stdlib `std/io`: file + network minimal abstractions aligned with current runtime APIs.  
    Done scope: `std/io` includes `out/out_ln/fail`, file APIs (`file/file_exists/file_read_all/file_write_all/mkdir_p`) and minimal TCP APIs (`net_addr/net_connect/net_send/net_recv/net_close`) with interpreter/C backend parity.
12. [x] Package management hardening: registry/git lock verification and clearer mismatch diagnostics.  
    Done scope: manifest dep resolution covers path/git/registry, writes `vox.lock` with source/rev/digest metadata, verifies lock consistency before build/test, and reports explicit mismatch/missing dependency diagnostics.

### Source Snapshot: `docs/internal/archive/23-backlog-next.md`

# Stage2 Backlog Next (P0/P1, no async/effect)

Status: **archived (closed)**.
Canonical closure + gate: `docs/internal/archive/25-p0p1-closure.md`, `make test-p0p1`.

Rule (historical): complete one item end-to-end (code + tests + docs + commit), then move to next item without leaving unresolved leftovers in the same scope.

## Items

1. [x] Generic variadic params end-to-end MVP: `xs: T...` typechecks/codegens as stable lowered form, with clear constraints and diagnostics.
2. [x] Generic type-param pack declaration usable end-to-end (remove skeleton rejections; define current semantics explicitly).
3. [x] Generic pack expansion design landing (call-site/type-site behavior and diagnostics consistency).
4. [x] Macro system strengthening: quote/unquote coverage parity for expression shapes and clearer unsupported diagnostics.
5. [x] Macro execution safety rails: deterministic expansion ordering and bounded recursion diagnostics hardening.
6. [x] Comptime evaluator parity pass: close remaining unsupported constant-expression gaps in the documented subset.
7. [x] IR semantic consistency pass: cast/compare edge behavior and verifier diagnostics alignment.
8. [x] Diagnostics layering pass: tighter primary span coverage for typecheck/irgen/macro errors.
9. [x] Testing framework UX pass: filtering/rerun/report fields consistency across engines.
10. [x] Stdlib generic cleanup: remove repetitive non-generic wrappers where generic APIs already exist.
11. [x] Package/dependency UX pass: lock mismatch diagnostics and remediation hints consistency.
12. [x] Stage2 documentation convergence: update language/spec/tooling docs to match implemented behavior exactly.
