# Stage2 Active Backlog (Canonical)

Status: active.

Purpose:
- This is the only active task list for stage2 language/tooling evolution.
- Closed batches must not be re-listed here.

Do-not-relist batches:
- `docs/archive/25-stage2-p0p1-closure.md` (items 1-12 closed)
- `docs/archive/26-stage2-closure-1-4-7-9.md` (items 1-4/7-9 closed)

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
    - Landed in `compiler/stage2/src/compiler/typecheck/tc_call.vox`, `compiler/stage2/src/compiler/irgen/gen_call_match.vox`, and `compiler/stage2/src/compiler/typecheck/consts.vox`, with compile/typecheck tests covering both runtime and const-call paths.
  - [x] A01-2 Heterogeneous type pack binding + true per-position substitution model.
    - [x] A01-2a Allow heterogeneous trailing explicit type args when pack is only a placeholder (not materialized in params/ret/variadic/bounds).
    - [x] A01-2b True per-position substitution model for materialized heterogeneous packs.
      - [x] Runtime call, const-eval call, and IRGen all support per-position materialization for params/ret/variadic type slots.
      - [x] Heterogeneous pack instantiation names are disambiguated (`pack`, `pack__1`, ...), avoiding monomorph collisions.
      - [x] Pack projection members (`Pack.N`) in materialization are supported across parse/typecheck/compile paths.
      - [x] Heterogeneous pack participation in bounds/where clauses is supported (trait bounds + comptime where reflect).
  - [x] A01-3 Value pack expansion and call-site lowering coherence.
    - Verified by pack-call/vec-call dual-mode tests in `compiler/stage2/src/compiler/typecheck/typecheck_test.vox` and `compiler/stage2/src/compiler/compile/compile_test.vox`.
  - Source: `docs/06-advanced-generics.md`.

- [ ] A02 String/borrow model convergence from transitional `String/str` aliasing to true `&str`/slice semantics.
  - [x] A02-1 Bare `str` is now rejected; use `String` for owned text and `&str`/`&'static str` for borrow-position text.
    - Covered in `compiler/stage2/src/compiler/typecheck/ctx.vox`, with compile/typecheck regressions in `compiler/stage2/src/compiler/typecheck/typecheck_test.vox` and `compiler/stage2/src/compiler/compile/compile_test.vox`.
  - [x] A02-2 `&mut`/`&'static mut` call arguments now require mutable place roots (local mutable var or member-chain rooted at one), across direct calls, variadic paths, generic calls, and method-sugar dispatch.
    - Covered in `compiler/stage2/src/compiler/typecheck/tc_call.vox`, with regressions in `compiler/stage2/src/compiler/typecheck/typecheck_test.vox` and `compiler/stage2/src/compiler/compile/compile_test.vox`.
  - [x] A02-3 Non-static `&T` call arguments now require place roots (identifier/member-chain rooted at local), across direct calls, generic calls, variadic paths, and method-sugar dispatch.
    - Covered in `compiler/stage2/src/compiler/typecheck/tc_call.vox`, with regressions in `compiler/stage2/src/compiler/typecheck/typecheck_test.vox` and `compiler/stage2/src/compiler/compile/compile_test.vox`.
  - [x] A02-4 `let` annotations with non-static borrow now validate initializer sources (`&T` requires place; `&mut T` requires mutable place).
    - Covered in `compiler/stage2/src/compiler/typecheck/tc_fn.vox`, with regressions in `compiler/stage2/src/compiler/typecheck/typecheck_test.vox` and `compiler/stage2/src/compiler/compile/compile_test.vox`.
  - Current gap: `&T`/`&str` are still transitional in type representation (borrow is tracked by signature metadata, not a first-class IR type).
  - Sources: `docs/13-standard-library.md`, `docs/21-stage1-compiler.md`, `docs/19-ir-spec.md`.

- [ ] A03 Runtime memory model convergence.
  - [x] A03-1 Runtime tracked allocations now support early release via `vox_rt_free`; non-escaping temp path buffers in `mkdir_p`/`walk_vox_files` are released eagerly instead of waiting for process exit.
    - Covered in `compiler/stage2/src/compiler/codegen/c_runtime.vox` and `compiler/stage2/src/compiler/codegen/c_emit_test.vox`.
  - [x] A03-2 `std/sync` handles now support explicit release (`mutex_drop`/`atomic_drop`) via new low-level drop intrinsics, reducing long-running tool memory retention without changing value semantics.
    - Covered in `compiler/stage2/src/compiler/typecheck/collect.vox`, `compiler/stage2/src/compiler/codegen/c_func.vox`, `compiler/stage2/src/compiler/codegen/c_runtime.vox`, `compiler/stage2/src/std/sync/sync.vox`, `compiler/stage2/src/compiler/codegen/c_emit_test.vox`, and `compiler/stage2/src/compiler/smoke_test.vox`.
  - Current gap: full ownership/drop semantics for general values/containers are still not finalized.
  - Source: `docs/21-stage1-compiler.md`.

- [x] A04 Package registry remoteization.
  - [x] A04-1 Registry dependencies now support remote git-backed registry roots (`git+...`/URL/`.git`) with clone/fetch cache under `.vox/deps/registry_remote`, then resolve `name/version` from cached checkout.
    - Covered in `compiler/stage2/src/main.vox` and selfhost integration `compiler/stage0/cmd/vox/stage1_integration_test.go` (`TestStage1BuildsStage2SupportsVersionDependencyFromRemoteRegistryGit`).
  - Source: `docs/11-package-management.md`.

### P1

- [x] A05 Macro system closure from MVP to stable full execution model (while keeping deterministic diagnostics).
  - [x] A05-1 Expression-site macro execution is now strictly typed: macro fns returning `AstStmt/AstItem` are rejected at expression macro call sites with deterministic diagnostics (`macro call requires AstExpr or AstBlock return type; got ...`).
    - Covered in `compiler/stage2/src/compiler/macroexpand/macroexpand.vox`, `compiler/stage2/src/compiler/macroexpand/user_macro_inline.vox`, and tests in `compiler/stage2/src/compiler/macroexpand/macroexpand_test.vox`.
  - [x] A05-2 Statement-site `name!(...)`/`compile!(...)` now accepts `AstStmt` return type (direct `ExprStmt` positions), while expression sites remain `AstExpr/AstBlock`-only.
  - Source: `docs/10-macro-system.md`.

- [x] A06 Diagnostics span coverage completion (remaining weak paths in typecheck/irgen).
  - [x] A06-1 Call-site diagnostics now emit concrete reasons for argument/type-arg failures instead of generic `typecheck failed` in common paths.
    - Covered in `compiler/stage2/src/compiler/typecheck/tc_call.vox`, `compiler/stage2/src/compiler/typecheck/typecheck_test.vox`, `compiler/stage2/src/compiler/compile/compile_test.vox`.
  - [x] A06-2 Reserved intrinsic/private prelude function call paths now report explicit type errors.
  - [x] A06-3 Member/struct-literal diagnostics upgraded from generic fallback to explicit unknown/private/type-mismatch messages.
    - Covered in `compiler/stage2/src/compiler/typecheck/tc_member.vox`, `compiler/stage2/src/compiler/typecheck/tc_struct_lit.vox`, `compiler/stage2/src/compiler/typecheck/tc_expr.vox` with paired tests in typecheck/compile suites.
  - [x] A06-4 Enum constructor diagnostics (`.Variant(...)` and `Enum.Variant(...)`) now emit explicit unknown-variant/arity/arg-mismatch/result-mismatch errors.
    - Covered in `compiler/stage2/src/compiler/typecheck/tc_call.vox` with paired tests in `compiler/stage2/src/compiler/typecheck/typecheck_test.vox` and `compiler/stage2/src/compiler/compile/compile_test.vox`.
  - Source: `docs/18-diagnostics.md`.

- [x] A07 Specialization rule strengthening (where-strength/ordering edge cases).
  - [x] A07-1 Reject impl head type params that are unconstrained by `for` target type; this removes ambiguous overlap that can be introduced only via extra impl-head params/bounds.
    - Covered in `compiler/stage2/src/compiler/typecheck/collect_traits_impls.vox` with paired tests in `compiler/stage2/src/compiler/typecheck/generics_test.vox` and `compiler/stage2/src/compiler/compile/compile_test.vox`.
  - Source: `docs/06-advanced-generics.md`.

## Deferred Scope

- [ ] D01 `--target` CLI, target triples, linker config, cross-compilation matrix.
  - Deferred by decision in this thread.
  - Source: `docs/16-platform-support.md`.

- [ ] D02 Thread-safety model (`Send/Sync` auto-derivation policy).
  - Source: `docs/08-thread-safety.md`.

- [ ] D03 Async model.
  - Source: `docs/09-async-model.md`.

- [ ] D04 Effect/resource system.
  - Source: `docs/00-overview.md`.

- [ ] D05 FFI/WASM detailed ABI/attribute model.
  - Source: `docs/17-ffi-interop.md`.
