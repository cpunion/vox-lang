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
- 2026-03-02: `int_cast_checked` pointer-width checks (`i64/u64 -> isize/usize`) now reuse prelude inline helpers (`vox_i64_outside_intptr_range`/`vox_u64_exceeds_uintptr_max`) instead of emitting per-cast `#if` blocks, reducing generated C preprocessor noise in large outputs.
- 2026-03-02: vec element typed ops now reuse IR elem type in codegen (`VecPush/Insert/Set/Pop/Remove/Get`) and emit typed load/store instead of per-access `memcpy`, while keeping bulk `VecExtend` copy on `memcpy`.
- 2026-03-02: unsigned integer `+/-/*/&/|/^` lowering now stays in native unsigned width (`u8/u16/u32/u64/usize`) instead of widening every operation to `uint64_t`, reducing redundant casts in generated C.
- 2026-03-02: `VecExtend` no longer emits runtime `elem_size` mismatch checks in codegen; IR verify now enforces both slots are declared `Vec` with identical element type, so the copy path remains lean and type-safe by construction.
- 2026-03-02: calls with dead destination temps are now emitted as plain call statements (no `tN = ...`) while preserving side effects, reducing generated C temp noise and `-Wunused-but-set-variable` warnings.
- 2026-03-02: dead-temp emission now uses exact read-set checks on kept instructions for call destinations, avoiding conservative `live_temps` false positives; selfhost C warning count (`-Wunused-but-set-variable`) dropped from 48 to 9 on macOS arm64.
- 2026-03-02: in `emit_driver_main`/`emit_test_main` modes, codegen now emits only call-graph-reachable functions (plus `ffi_export` roots), keeping library/no-driver emission unchanged; on macOS arm64 selfhost C dropped from 652,316 to 611,747 lines and `-Wunused-function` warnings from 806 to 428.
- 2026-03-02: vec allocation/grow scaffolding was hoisted into shared runtime-core inline helpers (`vox_vec_data_new` / `vox_vec_grow_to`) instead of re-emitting the same `malloc/realloc` growth loop at every vec op site; on macOS arm64 selfhost C dropped from 611,747 to 608,407 lines, size from 16,194,675 to 15,829,654 bytes, and `-Wunused-function` warnings from 428 to 3.
- 2026-03-02: build/test cache key derivation and cache-list formatting replaced several O(n^2) insertion-sort loops with shared O(n log n) sorting paths (`sort_source_files` / `sort_strings`) in `src/main_test_cache.vox`, reducing cache-bookkeeping overhead on large file/test sets without changing cache-key semantics.
- 2026-03-02: source-file ordering in build/test cache key derivation switched to in-place index shell-sort (`Vec[i32]` over source paths) so cache hashing avoids copying large `SourceFile.text` payloads while keeping deterministic order.
- 2026-03-02: package source-key aggregation now sorts by `(pkg,path)` index order first and then linearly folds digests per package, removing per-file binary-search/insert maintenance in `build_cache_pkg_source_keys_with_files`.
- 2026-03-02: removed unused one-hop cache wrappers (`format_*` and `discover_tests_cached*` passthroughs) in `src/main_test_cache.vox` to keep helper surface tighter and reduce generated compiler C function noise.
- 2026-03-02: removed additional cache passthrough wrappers (`test_build_cache_hit` / `write_test_build_cache`) and switched tests to direct `*_with_tests` calls, continuing forwarding-wrapper shrink in cache utilities.
- 2026-03-02: removed cache key passthrough wrappers (`test_build_cache_key` / `build_cache_pkg_source_keys` / `build_cache_key`) and switched cache tests to direct `*_with_files` + `.keys` / `.key` use.
- 2026-03-03: removed two remaining unused cache passthrough wrappers (`test_build_cache_contains_all` / `write_discover_tests_cache`) to keep cache helper surface minimal and reduce generated forwarding-function noise.
- 2026-03-03: codegen branch lowering now elides fallthrough jumps (`Br`/`CondBr` to the immediate next block) and only keeps labels for actually emitted jump targets, reducing redundant `goto`/label noise in generated C.
- 2026-03-03: jump-target label marking in C codegen now precomputes a sorted unique target set once per function and does binary-search lookup per block, removing repeated full-block scans in `emit_func` hot path.
- 2026-03-03: `CondBr` with identical targets now lowers to a single `goto` (or empty on fallthrough) instead of emitting redundant `if (...) goto X; else goto X;`.
- 2026-03-03: codegen now filters unreachable IR blocks (from function entry CFG reachability) before C emission, avoiding dead block body emission in generated C.
- 2026-03-03: vec op arg/index checks now reuse shared inline runtime-core helpers (`vox_vec_require_handle` / `vox_vec_checked_index` / `vox_vec_checked_insert_index`) instead of repeating the same guard branches at each vec op site; on macOS arm64 selfhost C dropped from 552,511 to 550,280 lines and `goto` count from 29,592 to 28,445.
- 2026-03-05: build cache scaffolding added `pkg-sem-v1` metadata read/write helpers (`build_sem_cache_hit` / `write_build_sem_cache`) and wired build flow to stamp semantic cache keys alongside existing `pkg-obj`/`link` keys, preparing phase-C semantic cache reuse without changing current compile behavior.
- 2026-03-05: build-mode cache key derivation now computes compile/link/sem keys in a single pass (`build_cache_key_triple_from_pkg_source_keys`) instead of separate pair+single derivations, reducing duplicated package-key hashing in cache bookkeeping paths.
- 2026-03-05: build mode now short-circuits `compile_query_shadow_prepare_for_target_with_files` when semantic cache (`pkg-sem`) already hits and query-shadow trace is off, avoiding one redundant query-shadow prepare pass on warm semantic-cache runs.
- 2026-03-05: test mode now also short-circuits `compile_query_shadow_prepare_for_target_with_files` when test-build cache already hits and query-shadow trace is off, avoiding redundant query-shadow prepare on warm test-cache runs.
- 2026-03-05: build mode cache/query key plumbing now reuses one `pkg keys` vector through cache-key derivation (`build_cache_key_triple_from_pkg_source_keys_keep`) and only consumes it for query dep hashes when needed, removing unconditional dual-copy (`query/cache`) bookkeeping in the combined cache+query-shadow path.
- 2026-03-05: build/test mode cache keys now include compiler revision (`version@channel`) in mode text, so compiler upgrades invalidate stale `pkg-sem`/`pkg-obj`/`link` cache entries instead of cross-version reusing old artifacts.
- 2026-03-05: build path now distinguishes `pkg-obj` full hit (`meta + .c + .o`) from C-source-only hit (`meta + .c`); when only `.o` is missing it reuses cached `.c` and recompiles object directly, skipping one full compile/typecheck/codegen pass.
- 2026-03-05: test path now adds the same partial-hit behavior: when test cache has valid `meta + .c + test-list` but the test binary is missing, it skips compile/typecheck and only re-runs `cc` from cached `.c` to rebuild the test executable.
- 2026-03-05: test mode now also derives/stamps `pkg-sem` keys and skips `compile_query_shadow_prepare_for_target_with_files` when semantic cache already hits (with query-shadow trace off), reusing precomputed package dep hashes instead of re-deriving them in the test query-shadow branch.
- 2026-03-05: test-build cache key derivation switched from per-file full-text re-hash (`test-build-cache-v4`) to package-source-key folding (`test-build-cache-v5`) and now computes test-cache key + sem key from one shared pkg-key pass in `vox test`, reducing duplicate cache hashing work on warm paths.
- 2026-03-05: removed now-unused `test_build_cache_key_with_files` wrapper and switched cache tests to direct `build_cache_pkg_source_keys_with_files(...).keys` + `test_build_cache_key_from_pkg_source_keys(...)`, continuing cache-helper wrapper shrink.
- 2026-03-05: test-cache subset check now treats cached compiled-test list as pre-sorted and avoids re-sorting `hay` on every state check (`test_build_cache_contains_all_sorted_hay_with_need`), trimming one redundant sort from warm-cache test selection.

## 5. Validation Gates

Required before each phase merge:
- `make fmt-check`
- `make test-active`
- cache hit/miss deterministic tests in `src/main_cache_test.vox`
- CI smoke on linux/darwin/windows + wasm
