# C Codegen Efficiency And Package Cache Plan

Status: active design note.

Date: 2026-02-27.

## 1. Snapshot And Measurements

Sample artifact:
- `target/debug/vox_rolling.c`
- size: ~14 MB
- lines: `611367`

Measured with `rg`/`wc`:
- `i32 to isize overflow` checks: `73`
- `if (tN < INTPTR_MIN || tN > INTPTR_MAX)` guards: `74`
- `memcpy(&vox_tmp_...)` occurrences: `3927`
- `vox_host_panic(...)` call sites: `507`

These numbers are from a selfhost build of the current compiler on macOS arm64.

## 2. Main Inefficiency Findings

1. Same-width overflow guards are still emitted in some paths.
- Example: `i32 -> isize` on 32-bit targets.
- Impact: noisy warnings and redundant branches.

2. Temporary bitcast scaffolding is overused.
- `memcpy(&vox_tmp_...)` appears very frequently for arithmetic/cast lowering.
- Impact: larger C output and extra operations for C compiler to optimize away.

3. Forwarding wrappers add C function noise.
- Simple pass-through helpers compile into many additional functions/labels.
- Impact: larger generated code and harder profiling.

4. Control-flow label expansion is very verbose.
- Large amount of `vox_blk_if_*`/`vox_blk_while_*` labels.
- Impact: C readability and optimization-debug friction.

## 3. Low-Risk Optimization Order

1. Cast/overflow elimination pass (highest ROI, lowest risk).
- Do not emit overflow check when source/target width and signedness prove safe by target pointer size.
- Start with `i32 <-> isize`, `usize <-> u32`, then expand.

2. Bitcast lowering simplification.
- Reduce `memcpy` use when direct cast is well-defined for the type pair.
- Keep `memcpy` only for strict aliasing-sensitive cases.

3. Forwarding-wrapper inlining in compiler source.
- Remove local one-hop wrappers and call canonical helpers directly.
- Continue the `vox/internal/*` reuse direction.

4. CFG pretty-pass for C emission (later).
- Merge trivial labels/blocks where it does not affect diagnostics/profiling mapping.

## 4. Package Compile + Build Cache Plan

Goal:
- Move from monolithic whole-package C emission/compile to package-unit incremental reuse.

### 4.1 Cache Units

1. Package semantic unit cache (`pkg-sem`).
- Key: package sources + resolved deps + target triple + emit options + compiler version hash.
- Payload: normalized typechecked package summary + exported symbol ABI signature digest.

2. Package C/object cache (`pkg-obj`).
- Key: `pkg-sem` key + codegen-relevant flags + C toolchain fingerprint.
- Payload: generated C and compiled object (`.o/.obj`) per package unit.

3. Link cache (`link`).
- Key: ordered object list digest + linker flags + target artifact kind.
- Payload: final binary/shared/static artifact.

### 4.2 Build Flow

1. Resolve dependency DAG from `vox.toml` + lockfile.
2. Topologically evaluate packages:
- hit `pkg-sem` -> skip package typecheck body work
- hit `pkg-obj` -> skip package C compile
3. Link stage:
- hit `link` -> copy cached artifact
- miss -> link and persist

### 4.3 Directory Layout (proposal)

- `target/cache/pkg-sem-v1/<key>/...`
- `target/cache/pkg-obj-v1/<key>/...`
- `target/cache/link-v1/<key>/...`

### 4.4 Invalidation Rules

Invalidate unit on any of:
- package source change
- transitive exported ABI digest change
- target/abi/artifact change
- compiler version/hash change
- C toolchain fingerprint change

### 4.5 Rollout Phases

1. Phase A (safe): link cache split + per-package key scaffolding (no behavior change).
2. Phase B: per-package C/object cache for dependencies first, root package last.
3. Phase C: semantic cache and incremental typecheck reuse.

### 4.6 Progress Notes

- 2026-02-28: link-cache hit check was decoupled from package C path existence.
- Current link hit condition is: `link` meta key match + linked binary artifact exists.
- Package object cache (`pkg-obj`) still requires C path + object + obj meta, unchanged.
- 2026-02-28: `std/prelude/string` pointer loops were rewritten to pointer-walk style (less inline `i32 -> isize` casts), reducing forced selfhost build warnings from 164 to 100 on macOS arm64.
- 2026-02-28: `rolling-selfhost` now prefers `target/debug/vox_rolling` as bootstrap when available, with automatic retry fallback to `target/bootstrap/vox_prev` on bootstrap build failure; this keeps daily builds on latest compiler while preserving recovery path.
- 2026-02-28: self-bootstrap cache key was stabilized for `target/debug/vox_rolling`; no-source-change runs now hit rolling selfhost cache (`rebuild: no`) instead of rebuilding on every run due to bootstrap mtime churn.
- 2026-02-28: `rolling-selfhost` rebuild check now short-circuits `FORCE_REBUILD` and missing-output cases before source hash computation, avoiding one unnecessary full-tree hash on rebuild-miss paths; bootstrap CLI probing also skips known modern compiler binary names (`vox_rolling`/`vox_tool`/`vox`) and probes only fallback/unknown names.

## 5. Validation Gates

Required before each phase merge:
- `make fmt-check`
- `make test-active`
- cache hit/miss deterministic tests in `src/main_cache_test.vox`
- CI smoke on linux/darwin/windows + wasm
