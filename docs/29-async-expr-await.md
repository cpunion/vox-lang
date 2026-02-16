# D03-5: await In General Expressions (try/match/macro args)

This doc tracks the next step after D03-4 (borrow constraints): removing remaining `await` placement
restrictions so async code is ergonomic without forcing users to rewrite everything into statement
form manually.

## Current Status (As Of v0.?)

Supported:

- `await` in `async fn` with state-machine lowering (frame + `poll`).
- `await` in nested `if/while` statement bodies.
- `await` in block expressions `{ ... }` (lifted to statement list).
- `await` in `if` expressions when the `if` expression is used in statement contexts:
  - `let` (requires explicit annotation type)
  - assignment
  - expression statement
  - `return`

Still rejected (compiler error from `async_norm`):

- `await` inside `try { ... }` expressions.
- `await` inside `match` expressions (in arms) when the `match` is used in arbitrary expression positions (non-statement contexts).
- `await` inside macro call arguments.

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

## Work Items (Queued)

1. `await` in `try { ... }`:
   - Preserve `?`-targeting semantics inside try scopes.
   - Ensure suspend/resume does not duplicate side effects.
   - Add tests covering `try { let x = fut.await?; x }` and nested `try`.

2. `await` in `if` expressions in arbitrary expression positions:
   - e.g. `foo(if c { a.await } else { b })`
   - Requires typed join slot (like current `@uninit()` path) without forcing user annotations.

3. `await` in `match` expressions (arms):
   - Full pattern binding + arm-local scoping.
   - Ensure captures and move tracking remain correct.

4. `await` in macro args:
   - Define a stable phase order: macroexpand -> async lowering -> IRGen.
   - Ensure diagnostics point to post-expand spans.

## Non-goals

- Effects/async IO runtime design (executor, waker integration beyond v0).
- Full generator syntax (`yield`) or async blocks (`async { ... }`) in this iteration.
