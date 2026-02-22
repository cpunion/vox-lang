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
  - [ ] A44-2 平台抽象收敛：`std/sys` 以平台原生 API 为主（Linux syscall/Unix/POSIX、Darwin、Windows API），避免将 `libc` 作为默认统一抽象层。
  - [ ] A44-3 编译器与标准库边界收敛：新增/迁移后不引入新的 `vox_builtin_*` / `vox_rt_*` 功能面；同等能力优先通过 `@ffi_import + @build` 在 `std/*` 实现（`@cfg` 仅保留测试）。
  - [ ] A44-4 回归与文档：补齐 `std/sys` + `std/fs` + FFI 相关测试，文档明确“何时需要 NUL 适配缓冲、何时直接 `ptr+len`”。
  - [ ] A44-5 最终目标：移除对 `libc` 的运行时依赖（含默认 `libc` I/O/socket/path 兜底路径）；平台实现以系统调用/原生 OS API 为主，仅在无法避免处保留最小兼容垫片并单独标注。
  - Source: `docs/internal/17-ffi-interop.md`, `docs/internal/13-standard-library.md`, `docs/internal/16-platform-support.md`.

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
