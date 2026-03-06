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
- 2026-03-05: removed additional unused/forwarding test-cache helpers (`test_build_cache_load_tests`, `test_build_cache_hit_with_tests`) and switched remaining tests to `test_build_cache_state_with_tests`, further shrinking cache utility surface.
- 2026-03-05: removed additional test-only cache key wrappers (`BuildCacheKeyPair`, `build_cache_key_pair_from_pkg_source_keys`, `build_cache_key_triple_from_pkg_source_keys`) and switched tests to `build_cache_key_triple_from_pkg_source_keys_keep(...).triple`.
- 2026-03-05: moved test-only helpers (`build_cache_pkg_keys`, `build_cache_obj_hit`) out of main cache helper module; tests now use local equivalents (`build_cache_test_pkg_keys_with_mode`, `build_cache_test_obj_hit`) so production cache surface keeps only runtime-used helpers.
- 2026-03-05: moved test-only direct key-derivation helpers (`build_cache_key_from_pkg_source_keys`, `test_build_cache_key_from_pkg_source_keys`) into cache tests and renamed to `build_cache_test_key_from_pkg_source_keys` / `cache_test_build_test_key_from_pkg_source_keys`, removing these utilities from production cache helper code.
- 2026-03-05: int-cast codegen now uses a pointer-width helper for `u32 -> isize` upper-bound checks (`vox_u32_exceeds_intptr_max`) so 64-bit targets compile the check away via preprocessor guard while 32-bit targets keep runtime overflow protection.
- 2026-03-05: int-cast codegen now also uses a pointer-width helper for `usize -> i64` upper-bound checks (`vox_uptr_exceeds_i64_max`) so 32-bit targets compile this check away while 64-bit targets keep the overflow guard.
- 2026-03-05: int-cast codegen now also uses a pointer-width helper for `isize -> u32` upper-bound checks (`vox_iptr_exceeds_u32_max`) so 32-bit targets compile this check away while 64-bit targets keep the overflow guard.
- 2026-03-05: semantic cache key derivation in build/test paths now hashes only `.vox` sources (`build_cache_sem_key_from_files`) while compile/link/test-build keys still hash mixed sources, so C-source-only edits no longer invalidate `pkg-sem` warm-path decisions.
- 2026-03-05: build/test hot paths now avoid deriving unused semantic/test keys in shared cache-key helpers (`build_cache_key_pair_from_pkg_source_keys_keep`, `test_build_key_from_pkg_source_keys_keep`), removing one redundant sem-hash loop from `vox build` and one redundant sem-hash loop from `vox test` cache setup.
- 2026-03-05: package-key scaffolding now has a split pass (`build_cache_pkg_sem_source_keys_with_files`) that emits both `keys` and `sem_keys` from one ordered scan; `vox build`/`vox test` now reuse `sem_keys` directly for `pkg-sem` key derivation, avoiding an extra sem-only rescan of source files on cache-enabled paths.
- 2026-03-05: `build_cache_pkg_source_keys_with_files` was switched back to a keys-only scan (no implicit sem key work), so query-shadow-only/non-cache paths avoid paying split-key computation overhead.
- 2026-03-05: warm build cache copy now memoizes the local `out.c` sync by compile key (`build_cache_output_copy_hit` + `.build.copy.key` sidecar) and skips cache->output C copy when unchanged, removing repeated large C-file copy work on hot cache hits.
- 2026-03-05: added src-only `.vox` walker (`walk_src_vox_files`) and switched build/dependency/stage1-load paths to it, avoiding redundant `tests/**` traversal on non-test flows; on macOS arm64 selfhost warm `vox build` dropped from ~0.75s to ~0.64s in local measurements.
- 2026-03-05: non-test source collection now uses compiler-internal `walk_src_non_test_vox_files` (`src/**` minus `*_test.vox`) across `vox build`, stage1 preload, dependency source load, and lock digest walk, removing unnecessary `src/*_test.vox` participation from non-test paths.
- 2026-03-05: `write_lockfile` now writes `vox.lock` only when content changes (`write_string_if_changed`), so repeated build/test runs with stable dependency graph no longer rewrite lockfile on every invocation.
- 2026-03-05: lock verification now returns computed `lock_deps` and build/test write path reuses them (`write_lockfile_from_lock_deps`), removing the second dependency digest pass that previously happened in `verify_lockfile_or_ok` + `write_lockfile`.
- 2026-03-05: test discovery cache miss path now reuses test-file text loaded during discover-key hashing (`discover_tests_cache_key_from_test_paths_keep`) for parse/discovery (`discover_tests_from_cached_test_files`), avoiding one redundant full read of every discovered test file on miss path.
- 2026-03-05: `load_files_with_cached` lookup switched from per-path linear scan of cached files to sorted-index + binary search (`source_file_path_index_order` + `find_cached_source_text`), reducing warm test-load matching from O(n*m) to O((n+m)log m) for large selected test sets.
- 2026-03-05: on test-discovery cache hit (`discover_tests_cached_with_paths`), preloaded file texts from discover-key hashing are now returned via `cached_files` instead of dropped, removing one redundant read of selected test sources in subsequent load phase.
- 2026-03-05: `source_file_path_index_order` now short-circuits when cached file paths are already non-decreasing, skipping shell-sort work on the common discover-key preloaded path set.
- 2026-03-05: cache string normalization paths now use `sort_strings_if_needed` (already-sorted fast path) for parse/format/key-derivation helpers in `src/main_test_cache.vox`, avoiding redundant sorts on canonical cache files while preserving deterministic ordering when input is unsorted.
- 2026-03-05: `load_files_with_cached` now uses a linear merge fast path when both selected `paths` and cached source paths are already sorted, reducing cached lookup matching from O(n log m) to O(n+m) on common warm test runs; it falls back to binary-search lookup when either side is unsorted.
- 2026-03-05: cache-hit parse helpers (`parse_test_build_cache_tests`, `parse_discover_tests_cache_list`) now preserve file order directly (no read-time sorting); deterministic sortedness remains enforced on cache-write path via `format_*_with_tests`.
- 2026-03-05: package source ordering helper (`source_file_order_by_pkg_path_with_files`) now short-circuits when source files are already non-decreasing by `(pkg,path)`, skipping shell-sort work on common preordered source lists while preserving the existing sort fallback for unsorted input.
- 2026-03-05: sem-key derivation/stamping is now gated by `query-shadow + build-cache` (`should_use_sem_cache`), so default cache-enabled runs reuse keys-only pkg hashing and avoid sem-key/meta work when query shadow is off.
- 2026-03-05: package owner classification for cache key scans now fast-paths common non-`pkg/*` roots (`src/`, `dep/`, `tests/`, `std/`, `examples/`, `target/`) to `main`, avoiding unnecessary `mod_path_from_file_path` work on most build/test source lists.
- 2026-03-05: package owner classification now also fast-paths dependency virtual paths (`<dep_name>/src/...`) to `pkg/<dep_name>`, so dependency-source cache scans avoid per-file `mod_path_from_file_path` for the common dep layout.
- 2026-03-05: cache metadata/list writers now use `write_string_if_changed` (`build/test cache key sidecars`, `discover-tests cache list/key`, and `test-build cache list/key`) to avoid rewriting unchanged cache files on hot runs.
- 2026-03-05: moved test-only triple-key derivation helper/types (`BuildCacheKeyTriple*` + `build_cache_key_triple_from_pkg_source_keys_keep`) out of production cache helper module into cache tests (`BuildCacheTestKeyTriple*`), further shrinking runtime cache surface and generated forwarding noise.
- 2026-03-05: moved additional test-only wrappers/derivers out of production cache helper module (`build_cache_sem_key_from_files*`, `discover_tests_cache_key_from_test_paths`); cache tests now use local helpers over existing keep-APIs, further trimming production helper surface.
- 2026-03-05: moved test-only `test_build_sem_keys_from_pkg_source_keys_keep` (`TestBuildSemKeysKeep`) out of production cache helper module and into cache tests, keeping test-sem-key derivation coverage while reducing production helper count.
- 2026-03-05: inlined single-use `build_cache_sem_out_for_key` into `build_cache_sem_meta_path_for_key`, removing another one-hop helper from production cache utilities.
- 2026-03-05: inlined two additional single-use cache helpers (`test_build_cache_load_tests_for_c`, `write_discover_tests_cache_with_tests`) into their callers, keeping behavior unchanged while trimming production helper/function count.
- 2026-03-05: folded `strings_are_non_decreasing` into `sort_strings_if_needed` (single caller), removing one extra helper while preserving sorted-fast-path behavior.
- 2026-03-05: inlined package-digest initializer helper (`build_cache_pkg_digest_init` + `BuildCachePkgDigest`) at the two package-key accumulation sites, removing another helper/type pair from production cache utilities.
- 2026-03-05: removed shared string-copy helper/type (`StringCopyResult`, `copy_strings_keep`) and inlined copy loops at the three use sites (`format_test_build_cache_tests_with_tests`, `test_build_cache_contains_all_sorted_hay_with_need`, `format_discover_tests_cache_list_with_tests`), reducing helper surface without behavior change.
- 2026-03-05: inlined test-build cache path wrappers (`test_build_cache_meta_path`, `test_build_cache_tests_path`, `test_build_cache_root_for_key`) into direct callers (`test_build_cache_out_for_key`, `test_build_cache_cpath_for_key`, cache state/write paths), further reducing one-hop helper surface.
- 2026-03-06: removed single-use test-list fast wrapper (`discover_tests_cached_fast_with_paths`) and inlined the same key/list cache hit path directly at the `vox test --list` call site in `main.vox`, keeping behavior unchanged while shrinking cache helper surface.
- 2026-03-06: in `vox/query`, inlined single-use shadow-root helper (`parse_load_shadow_root`) into `parse_load_shadow_meta_path`, removing one extra helper frame in query-shadow cache path utilities.
- 2026-03-06: in `vox/query`, removed single-use byte-hash helper (`hash_byte`) and inlined the same FNV step directly in `hash_text`, trimming one helper frame from parse-load key hashing utilities.
- 2026-03-06: in `vox/query`, inlined single-use file-order helper (`file_indices_by_path`) into `hash_parse_load_files`, keeping deterministic file hashing while reducing one-hop helper indirection.
- 2026-03-06: in `vox/query`, removed local dep comparator helper (`dep_less`) and inlined equivalent comparison in `dep_indices_sorted`, reducing helper calls in parse-load dependency sort while preserving key order semantics.
- 2026-03-06: in `vox/query`, removed single-use dep-order helper (`dep_indices_sorted`) and inlined its deterministic sort logic directly in `hash_parse_load_deps`, further shrinking parse-load hash helper surface.
- 2026-03-06: in `vox/query`, removed single-use switch-hash helper (`hash_parse_load_switches`) and inlined its hashing sequence directly in `parse_load_key`, keeping key schema unchanged while reducing helper indirection.
- 2026-03-06: in `vox/query`, removed single-use dep-hash helper (`hash_parse_load_deps`) and inlined its deterministic dependency ordering + hash fold directly in `parse_load_key`, preserving parse-load key semantics while reducing helper layering.
- 2026-03-06: in `vox/query`, removed single-use file-hash helper (`hash_parse_load_files`) and inlined its deterministic file ordering + hash fold directly in `parse_load_key`, further shrinking parse-load key helper layering without changing key schema.
- 2026-03-06: in `vox/query`, removed local bool-string helper (`bool_text`) and inlined boolean-text selection in `parse_load_key`, keeping key material unchanged while trimming another helper indirection.
- 2026-03-06: in `vox/query` parse-load key hashing, file ordering now sorts file indices directly by `files[i].path` (shell-sort over index vector) instead of building a separate copied `file_paths` vector first, reducing one allocation/copy pass while preserving deterministic order.
- 2026-03-06: in `vox/query`, removed single-use feature-order helper (`sort_string_indices`) and inlined deterministic feature-switch sorting directly in `parse_load_key`, preserving key schema while reducing helper layering.
- 2026-03-06: in `vox/query`, added a sorted-fast-path for `feature_switches` in `parse_load_key`: when switches are already non-decreasing, hash directly in-place and skip index allocation/sort; unsorted input still falls back to the same deterministic shell-sort order.
- 2026-03-06: in `vox/query`, added sorted-fast-paths for dependency/file hashing in `parse_load_key`: when `deps`/`files` are already in canonical order, hash directly without building index-order vectors or running shell-sort; unsorted inputs still use the same deterministic fallback ordering.
- 2026-03-06: removed full `SourceFile` copy in compile query-shadow prepare path: added `parse_load_shadow_prepare_keep_files` in `vox/query` and switched `compile_query_shadow_prepare_for_target_with_files` to consume+return the original `files` vector while deriving the parse-load key, avoiding duplicate source-text copy when query-shadow prepare runs.
- 2026-03-06: unified query-shadow dependency hash type across `main`/`compile` to `q.ParseLoadDepHash` and removed compile-side conversion loop (`CompileQueryDepHash` -> `ParseLoadDepHash`), trimming one per-prepare vector remap on build/test query-shadow paths.
- 2026-03-06: query-shadow prepare now skips parse-load shadow-hit metadata I/O when trace is off (`trace_predicted_hit=false`): build/test paths set this from `query_shadow_trace`, so default runs derive key without `fs.exists/read_to_string` hit-probe cost while trace-on behavior remains unchanged.
- 2026-03-06: cached loop bounds (`len()`) in `vox/query` parse-load key hot loops (`hash_text`, feature/dep/file scans) to avoid repeated `len()` calls inside tight loops while preserving key material and ordering.
- 2026-03-06: unified compile-layer query-shadow state type to `q.ParseLoadShadowState` and removed compile-side state remap/copy (`CompileQueryShadowState`), so prepare/trace/write paths now pass query state through directly.
- 2026-03-06: cached shell-sort loop bounds in `vox/query` parse-load key fallback ordering (`feature_order`/`dep_order`/`file_order`) by reusing precomputed `feature_n`/`dep_n`/`file_n`, removing repeated `Vec.len()` calls in hot-key derivation loops while preserving deterministic order.
- 2026-03-06: build-mode query-shadow prepare skip now aligns with test mode and treats compile-C cache hit as a skip condition (`compile_c_cache_hit || sem_cache_hit`) when trace is off, avoiding unnecessary parse-load key/hash work on warm C-cache builds.
- 2026-03-06: removed compile-layer one-hop trace wrapper (`compile_query_shadow_trace_line`) and switched `main` to call `q.parse_load_shadow_trace_line` directly, trimming one forwarding API while keeping trace output unchanged.
- 2026-03-06: `compile_query_shadow_prepare_for_target` now reuses `compile_query_shadow_prepare_for_target_with_files` and returns only `.state`, removing duplicate trace/no-trace switch and parse-load switch-construction logic while preserving external API behavior.
- 2026-03-06: `compile_query_shadow_prepare_for_target_with_files` now builds `ParseLoadKeySwitches` once via shared helper (`compile_query_shadow_switches`) and reuses it in trace/no-trace branches, trimming duplicated switch-construction code while preserving query-shadow behavior.
- 2026-03-06: narrowed `compile_query_shadow_switches` inputs to only switch-related fields (instead of whole `CompileQueryShadowOptions`), avoiding potential by-value carry/copy of unused `dep_hashes` while keeping query-shadow key material unchanged.
- 2026-03-06: `parse_load_shadow_prepare` now computes key+predicted-hit directly from `parse_load_key` instead of routing through `parse_load_shadow_prepare_keep_files`, avoiding unnecessary keep-files result packing on state-only callers.
- 2026-03-06: removed compile-layer one-hop write wrapper (`compile_query_shadow_write`) and switched `main` build/test write points to direct `q.parse_load_shadow_write` with existing empty-key guard, trimming another forwarding API without behavior change.
- 2026-03-06: removed unused state-only query-shadow prepare APIs (`parse_load_shadow_prepare`, `parse_load_shadow_prepare_no_hit`) and kept `parse_load_shadow_prepare_keep_files*` as the single active prepare surface, reducing dead exported helper surface in `vox/query`.
- 2026-03-06: removed compile-layer state-only query-shadow prepare wrapper (`compile_query_shadow_prepare_for_target`) and switched compile tests to consume `.state` from `compile_query_shadow_prepare_for_target_with_files`, keeping behavior while shrinking dead forwarding API surface.
- 2026-03-06: inlined compile-layer single-use query-shadow switch builder (`compile_query_shadow_switches`) and its local driver-kind text helper directly into `compile_query_shadow_prepare_for_target_with_files`, reducing one-hop helper surface while preserving key-material construction.
- 2026-03-06: removed `vox/query` single-hop key wrapper (`parse_load_key`) and switched query tests to read `.key` from `parse_load_key_keep_files`, keeping deterministic key assertions while shrinking dead exported helper surface.
- 2026-03-06: removed compile-layer unused `compile_query_shadow_disabled` helper (single test caller) and inlined equivalent disabled-options literal in `query_shadow_test`, trimming another dead forwarding helper.
- 2026-03-06: removed two dead ptr-bits wrapper APIs in `vox/compile` (`compile_main_text_to_c_for_ptr_bits`, `compile_files_to_c_for_ptr_bits`) that had no in-repo callers; canonical target/profile entrypoints remain unchanged.
- 2026-03-06: removed compile-layer profile ptr-bits wrapper (`compile_files_to_c_profile_for_ptr_bits`) and switched compile profile test to call `compile_files_to_c_profile_for_target(..., \"unknown\", \"unknown\")` directly, reducing one-hop wrapper surface.
- 2026-03-06: in `vox/query`, removed single-use `parse_load_shadow_meta_path` helper and inlined equivalent cache meta-path construction in `parse_load_shadow_predict_hit` / `parse_load_shadow_write`, trimming helper surface without behavior change.
- 2026-03-06: removed compile-layer target wrapper (`compile_files_to_c_for_target`) and switched remaining compile tests to use `compile_files_to_c_profile_for_target(...).r` directly, reducing one-hop wrapper surface while keeping behavior unchanged.

## 5. Validation Gates

Required before each phase merge:
- `make fmt-check`
- `make test-active`
- cache hit/miss deterministic tests in `src/main_cache_test.vox`
- CI smoke on linux/darwin/windows + wasm
