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

- [ ] A01 Real generic pack expansion (type/value packs), not declaration-only.
  - [x] A01-1 Trailing explicit type args can bind a single trailing type pack.
    - Landed in `compiler/stage2/src/compiler/typecheck/tc_call.vox`, `compiler/stage2/src/compiler/irgen/gen_call_match.vox`, and `compiler/stage2/src/compiler/typecheck/consts.vox`, with compile/typecheck tests covering both runtime and const-call paths.
  - [ ] A01-2 Heterogeneous type pack binding + true per-position substitution model.
    - [x] A01-2a Allow heterogeneous trailing explicit type args when pack is only a placeholder (not materialized in params/ret/variadic/bounds).
    - [ ] A01-2b True per-position substitution model for materialized heterogeneous packs.
  - [ ] A01-3 Value pack expansion and call-site lowering coherence.
  - Source: `docs/06-advanced-generics.md`.

- [ ] A02 String/borrow model convergence from transitional `String/str` aliasing to true `&str`/slice semantics.
  - Current gap: `str` and `&T` are still transitional mappings in type layer.
  - Sources: `docs/13-standard-library.md`, `docs/21-stage1-compiler.md`, `docs/19-ir-spec.md`.

- [ ] A03 Runtime memory model convergence.
  - Current gap: process-lifetime cleanup exists, but full ownership/drop/release semantics are not finalized.
  - Source: `docs/21-stage1-compiler.md`.

- [ ] A04 Package registry remoteization.
  - Current gap: registry dependency currently resolves from local cache only.
  - Source: `docs/11-package-management.md`.

### P1

- [ ] A05 Macro system closure from MVP to stable full execution model (while keeping deterministic diagnostics).
  - Source: `docs/10-macro-system.md`.

- [ ] A06 Diagnostics span coverage completion (remaining weak paths in typecheck/irgen).
  - Source: `docs/18-diagnostics.md`.

- [ ] A07 Specialization rule strengthening (where-strength/ordering edge cases).
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
