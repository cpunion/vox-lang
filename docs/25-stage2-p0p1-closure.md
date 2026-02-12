# Stage2 P0/P1 Closure (Items 1-12)

Status: **closed**.

Single acceptance gate:

```bash
make test-stage2-p0p1
```

This gate is wired in `Makefile` and `scripts/ci/verify-stage2-p0p1.sh`.

## Why The 1-12 List Kept Reappearing

1. There was no single closure command; completion was spread across `docs/22-stage2-backlog.md`, `docs/23-stage2-backlog-next.md`, and several test suites.
2. Follow-up planning mixed “already-closed 1-12” with other roadmap work, so the same scope was repeatedly re-listed.
3. No explicit “archived + verified by command” convention existed for this batch.

## Item -> Evidence

1. Parser trailing comma completeness for generic call args.
   Evidence: `compiler/stage2/src/compiler/parse/parse_test.vox` (`test_parse_call_explicit_type_args_trailing_comma`, `test_parse_call_explicit_const_args_trailing_comma`, `test_parse_macro_call_with_explicit_type_args_trailing_comma`).
2. Macroexpand inline fallback reason diagnostics.
   Evidence: `compiler/stage2/src/compiler/macroexpand/macroexpand_test.vox` (`test_macroexpand_user_macro_sugar_lowers_to_call`, `test_macroexpand_user_macro_inline_member_fallback_for_cross_module_ident_smoke`).
3. Macro execution v1 (Ast* returning function-like macros).
   Evidence: `compiler/stage2/src/compiler/macroexpand/macroexpand_test.vox` (`test_macroexpand_ast_expr_macro_fn_exec_and_strip_smoke`, `test_macroexpand_compile_bang_executes_ast_expr_fn_call_smoke`).
4. `quote`/`unquote` MVP and `$x` interpolation.
   Evidence: `compiler/stage2/src/compiler/macroexpand/macroexpand_test.vox` (`test_macroexpand_quote_unquote_mvp_smoke`, `test_macroexpand_quote_expr_dollar_surface_syntax_smoke`, `test_macroexpand_quote_unquote_if_expr_shape_smoke`).
5. Comptime evaluator subset expansion.
   Evidence: `compiler/stage2/src/compiler/compile/compile_test.vox` (`test_compile_const_fn_call_with_pure_member_methods_smoke`, `test_compile_const_fn_call_with_escape_and_float_to_string_smoke`, `test_compile_const_fn_call_with_string_index_methods_smoke`).
6. Generic specialization diagnostics hardening.
   Evidence: `compiler/stage2/src/compiler/typecheck/generics_test.vox` and `compiler/stage2/src/compiler/compile/compile_test.vox` specialization overlap tests.
7. Generic packs/variadics MVP and diagnostics.
   Evidence: `compiler/stage2/src/compiler/parse/parse_test.vox` (`test_parse_fn_type_param_pack_and_variadic_param`) and `compiler/stage2/src/compiler/typecheck/typecheck_test.vox` pack/variadic acceptance + rejection tests.
8. Diagnostics upgrade (rune-aware column + tighter spans).
   Evidence: `compiler/stage2/src/compiler/parse/parse_test.vox` (`test_parse_error_column_uses_rune_index`) and `compiler/stage2/src/compiler/compile/compile_test.vox` span/rune diagnostics tests.
9. Testing framework JSON/rerun metadata pipeline.
   Evidence: `compiler/stage0/cmd/vox/main_test.go` JSON/rerun/unit tests and `compiler/stage0/cmd/vox/stage1_integration_test.go` `TestStage1BuildsStage2AndRunsStage2Tests` assertions.
10. `std/sync` generic `Mutex[T]`/`Atomic[T]` runtime semantics.
    Evidence: `compiler/stage2/src/compiler/smoke_test.vox` sync runtime tests and `compiler/stage2/src/compiler/compile/compile_test.vox` std sync compile tests.
11. `std/io` file/network minimal abstraction.
    Evidence: `compiler/stage2/src/compiler/compile/compile_test.vox` (`test_compile_std_io_smoke`, `test_compile_std_io_net_intrinsics_smoke`).
12. Package management hardening (lock/git/registry + diagnostics).
    Evidence: `compiler/stage0/cmd/vox/stage1_integration_test.go` (`TestStage1BuildPkgWritesLockfile`, `TestStage1BuildPkgFailsWhenLockDigestMismatch`, `TestStage1BuildPkgSupportsVersionDependencyFromRegistryCache`, `TestStage1BuildPkgSupportsGitDependency`, `TestStage1BuildsStage2LockMismatchDiagnosticsConsistentAcrossBuildAndTest`).

## Policy

- `docs/22-stage2-backlog.md` and `docs/23-stage2-backlog-next.md` are archived for this batch.
- If the gate passes, 1-12 stay closed.
- New work must open as a new list, not by re-listing 1-12.
