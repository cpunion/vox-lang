# Stage2 Active Backlog (Canonical)

Status: active.

Purpose:
- This is the only active task list for compiler language/tooling evolution.
- Closed batches must not be re-listed here.

Do-not-relist batches:
- `docs/archive/25-p0p1-closure.md` (items 1-12 closed)
- `docs/archive/26-closure-1-4-7-9.md` (items 1-4/7-9 closed)

## Why Tasks Repeated Before

1. Multiple backlog files existed, but no single canonical active list.
2. “Spec draft/deferred” lines were mixed into implementation TODO scans.
3. “完成一批” lacked a stable ID system, so similar items reappeared with new wording.

Governance from now on:
1. Every active item has a stable ID (`Axx`).
2. Completion requires code + tests + docs + commit, then mark `[x]` here.
3. Deferred items stay in Deferred and are not counted as active scope.
4. If a closed item must reopen, add explicit reason + regression evidence.

## Active Scope (non-deferred)

### P0

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
  - Source: `docs/06-advanced-generics.md`.

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
  - Sources: `docs/13-standard-library.md`, `docs/archive/21-stage1-compiler.md`, `docs/19-ir-spec.md`.

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
  - Source: `docs/archive/21-stage1-compiler.md`.

- [x] A04 Package registry remoteization.
  - [x] A04-1 Registry dependencies now support remote git-backed registry roots (`git+...`/URL/`.git`) with clone/fetch cache under `.vox/deps/registry_remote`, then resolve `name/version` from cached checkout.
    - Covered in `src/main.vox` and selfhost integration `archive/stage0-stage1:compiler/stage0/cmd/vox/stage1_integration_test.go` (`TestStage1BuildsStage2SupportsVersionDependencyFromRemoteRegistryGit`).
  - Source: `docs/11-package-management.md`.

### P1

- [x] A05 Macro system closure from MVP to stable full execution model (while keeping deterministic diagnostics).
  - [x] A05-1 Expression-site macro execution is now strictly typed: macro fns returning `AstStmt/AstItem` are rejected at expression macro call sites with deterministic diagnostics (`macro call requires AstExpr or AstBlock return type; got ...`).
    - Covered in `src/vox/macroexpand/macroexpand.vox`, `src/vox/macroexpand/user_macro_inline.vox`, and tests in `src/vox/macroexpand/macroexpand_test.vox`.
  - [x] A05-2 Statement-site `name!(...)`/`compile!(...)` now accepts `AstStmt` return type (direct `ExprStmt` positions), while expression sites remain `AstExpr/AstBlock`-only.
  - Source: `docs/10-macro-system.md`.

- [x] A06 Diagnostics span coverage completion (remaining weak paths in typecheck/irgen).
  - [x] A06-1 Call-site diagnostics now emit concrete reasons for argument/type-arg failures instead of generic `typecheck failed` in common paths.
    - Covered in `src/vox/typecheck/tc_call.vox`, `src/vox/typecheck/typecheck_test.vox`, `src/vox/compile/compile_test.vox`.
  - [x] A06-2 Reserved intrinsic/private prelude function call paths now report explicit type errors.
  - [x] A06-3 Member/struct-literal diagnostics upgraded from generic fallback to explicit unknown/private/type-mismatch messages.
    - Covered in `src/vox/typecheck/tc_member.vox`, `src/vox/typecheck/tc_struct_lit.vox`, `src/vox/typecheck/tc_expr.vox` with paired tests in typecheck/compile suites.
  - [x] A06-4 Enum constructor diagnostics (`.Variant(...)` and `Enum.Variant(...)`) now emit explicit unknown-variant/arity/arg-mismatch/result-mismatch errors.
    - Covered in `src/vox/typecheck/tc_call.vox` with paired tests in `src/vox/typecheck/typecheck_test.vox` and `src/vox/compile/compile_test.vox`.
  - Source: `docs/18-diagnostics.md`.

- [x] A07 Specialization rule strengthening (where-strength/ordering edge cases).
  - [x] A07-1 Reject impl head type params that are unconstrained by `for` target type; this removes ambiguous overlap that can be introduced only via extra impl-head params/bounds.
    - Covered in `src/vox/typecheck/collect_traits_impls.vox` with paired tests in `src/vox/typecheck/generics_test.vox` and `src/vox/compile/compile_test.vox`.
  - Source: `docs/06-advanced-generics.md`.

## Deferred Scope

- [ ] D01 `--target` CLI, target triples, linker config, cross-compilation matrix.
  - Deferred by decision in this thread.
  - Source: `docs/16-platform-support.md`.

- [x] D02 Thread-safety model (`Send/Sync` auto-derivation policy).
  - Stage2 baseline landed: marker traits in `std/prelude` + auto-derivation for scalars/String/Vec/Range/struct/enum; type params still require explicit bounds.
  - Source: `docs/08-thread-safety.md`.

- [ ] D03 Async model.
  - Source: `docs/09-async-model.md`.

- [ ] D04 Effect/resource system.
  - Source: `docs/00-overview.md`.

- [ ] D05 FFI/WASM detailed ABI/attribute model.
  - Source: `docs/17-ffi-interop.md`.

- [x] D06 First-class borrow IR/type representation (`&T`/`&str` non-erased types, borrow-aware IR ops).
  - [x] D06-1 Type-pool level borrow representation landed: `ir::TyKind.Ref` + `resolve_type` preserves `&T/&mut T/&'static T/&'static mut T` and reflection (`@type_name/@type`) can observe borrow shape.
  - [x] D06-2 Stage2 bootstrap boundary updated: irgen now preserves `Ref` in IR signatures/slots/calls, while `Range` continues to lower to base + `range_check` for v0 stability.
  - [x] D06-3 Borrow-aware IR/codegen landed: codegen `Ref` transparent type mapping + compare/nominal-eq borrow-aware unwrapping, with regression tests for IR signature preservation and `&str` compare codegen.
  - Extracted from A02 closure note.
  - Source: `docs/19-ir-spec.md`, `docs/13-standard-library.md`.

- [ ] D07 Full ownership/move/drop semantics for general values/containers.
  - [x] D07-1 Remove bootstrap-safe `std/collections/map` fallback: switch to direct `Vec.set/remove/clear` implementation and keep `stage1 -> compiler` gate green.
  - [x] D07-2 Container-level deterministic release model (Vec/String/Map) that is alias-safe under current value-copy semantics.
    - [x] D07-2a Deep-clone baseline landed: `Clone` trait + `impl[T: Clone] Clone for Vec[T]` + `impl[K: Eq + Clone, V: Clone] Clone for Map[K,V]` for explicit non-aliasing copy paths.
    - [x] D07-2b Add deterministic release semantics compatible with the current value-copy model (no UAF on alias copies).
      - Landed baseline: prelude `Release` trait + `release(String)`/`release_vec(Vec[T])`/`Map.release()` reset APIs; release is alias-safe and idempotent, while physical shared-storage reclaim remains deferred to D07-3 ownership/move/drop model.
  - [ ] D07-3 Language-level ownership/move/drop rules and diagnostics (no-UAF contract) across function boundaries and aggregates.
    - [x] D07-3a Release API rebind enforcement: expression-statement `release` calls are rejected (`release call result must be assigned back`) to avoid silent non-rebinding misuse.
    - [x] D07-3b Minimal move-after-release diagnostics: values consumed by `release` are marked moved; later reads error as `use of moved value: <name>`, while `x = release(x)` remains a valid self-rebind path.
    - [x] D07-3c Move-state propagation baseline in control flow: `block/if/while` now conservatively propagate moved flags for outer locals, so branch/loop release paths are visible to later reads.
    - [x] D07-3d Aggregate-root move propagation for release paths: `release(x.field)` / `x.field.release()` now mark root `x` as moved (conservative no-UAF baseline under current copy semantics).
    - [x] D07-3e Member-chain moved-value diagnostics propagation: receiver member/call paths now preserve upstream `use of moved value` diagnostics instead of degrading to enum/path fallback errors.
  - Extracted from A03 closure note.
  - Source: `docs/archive/21-stage1-compiler.md`.
