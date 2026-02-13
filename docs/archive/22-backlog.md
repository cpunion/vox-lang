# Stage2 Backlog (1-12)

Status: **archived (closed)**.
Canonical closure + gate: `docs/archive/25-p0p1-closure.md`, `make test-p0p1`.

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
