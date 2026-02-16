# D03-5: await In General Expressions (try/match/macro args)

This doc tracks the next step after D03-4 (borrow constraints): removing remaining `await` placement
restrictions so async code is ergonomic without forcing users to rewrite everything into statement
form manually.

## Current Status (As Of v0.2.x)

Supported:

- `await` in `async fn` with state-machine lowering (frame + `poll`).
- `await` in nested `if/while` statement bodies.
- `await` in block expressions `{ ... }` (lifted to statement list).
- `await` in `if` expressions when the `if` expression is used in statement contexts:
  - `let` (requires explicit annotation type)
  - assignment
  - expression statement
  - `return`
- `await` in general nested expression control-flow:
  - `try { ... }` expressions
  - `match` expressions (including arm bodies)
  - macro call arguments

This closes D03-5 scope from `docs/27-active-backlog.md`.

## Why These Are Hard In The Current Pipeline

The current async lowering pipeline is intentionally simple:

1. Normalize/lift `await` out of expression trees into the surrounding statement list.
2. Build a statement-level async CFG (splitting on `await` statements).
3. Compute captures (liveness) across resume points and store them in the async frame.
4. IRGen generates the `poll` function as a state machine.

This works well when all suspension points are statement-level. But:

- `try { ... }` introduces an *expression-scoped* propagation target for `?`.
- `match` introduces multi-branch joins and pattern bindings (arm-local locals).
- Macro args require a stable ordering between macroexpand and async normalization.

Solving these correctly requires a more general lowering that can split control-flow within expression
evaluation while preserving:

- evaluation order (left-to-right),
- single-evaluation (no side-effect re-execution),
- `?` target semantics (`try` scope vs function return),
- pattern-binding scoping in `match`.

## Proposed Direction

Introduce a *typed* async expression lowering stage used only for `async fn` bodies:

- Input: AST + type info per expression (post-typecheck).
- Output: a lowered "async HIR" statement list where:
  - control-flow joins are explicit,
  - all suspension points are explicit `AwaitStmt` nodes,
  - `try` scopes are explicit regions with a join target,
  - `match` is lowered into explicit arm blocks with join slots.

This lowered form becomes the input to the existing async CFG builder and capture analysis.

## Verification

- Typecheck tests:
  - `test_typecheck_async_fn_allows_await_inside_try_block_smoke`
  - `test_typecheck_async_fn_allows_await_in_nested_match_expr_smoke`
  - `test_typecheck_async_fn_allows_await_in_macro_arg_smoke`
- Compile tests:
  - `test_compile_async_fn_allows_await_inside_try_block_smoke`
  - `test_compile_async_nested_match_expr_with_await_smoke`
  - `test_compile_async_fn_allows_await_in_macro_arg_smoke`

## Remaining Async Work (Outside D03-5)

1. Generic async instantiation for const/type-pack parameter shapes (`async_inst` still rejects these).
2. Runtime executor integration for async entrypoints (`async main` / async tests).

## Completed Since This Doc

- IRGen now supports `await` lowering when `std/async::Future::poll` method signatures are generic:
  - generic method instantiation is materialized and queued via `PendingInst`,
  - default const params (when present) are applied for monomorphized poll calls,
  - async await paths are covered by typecheck/irgen/compile smoke tests.

## Non-goals

- Effects/async IO runtime design (executor, waker integration beyond v0).
- Full generator syntax (`yield`) or async blocks (`async { ... }`) in this iteration.
