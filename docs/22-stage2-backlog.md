# Stage2 Backlog (1-12)

This file is the active burn-down list for stage2.  
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
8. [ ] Diagnostics upgrade: rune-aware column mapping and tighter span for type/const/macro errors.
9. [ ] Testing framework upgrade: richer `--json` payload and stable rerun metadata pipeline.
10. [ ] Stdlib `std/sync`: generic `Mutex[T]` / `Atomic[T]` runtime-backed semantics on stage2.
11. [ ] Stdlib `std/io`: file + network minimal abstractions aligned with current runtime APIs.
12. [ ] Package management hardening: registry/git lock verification and clearer mismatch diagnostics.
