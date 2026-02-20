# Async and Await

## Scope

Defines async function syntax, await forms, async trait methods, and user-visible lowering behavior.

Coverage IDs: `S501`, `S502`, `S503`, `S504`, `S505`.

## Grammar (Simplified)

```vox
AsyncFnDecl
  := "async" "fn" Ident Signature Block

AwaitExpr
  := Expr ".await"
   | "await" Expr   // compatibility form
```

Async method declarations in traits are supported:

```vox
trait T {
  async fn m(x: Self) -> Ret;
}
```

## Semantics

- `async fn` is lowered to a future-like state machine.
- `expr.await` polls until ready and yields the output value.
- `await expr` is accepted as compatibility syntax; `expr.await` is preferred.
- Await is supported in expression contexts covered by lowering, including:
  - statement-level awaits,
  - `if`/`match` expression branches,
  - try/`?` composition (`expr.await?`),
  - macro argument positions after normalization.

## Await Operand Model

`await` accepts either:

- Poll-like enum shape with variants `{ Pending, Ready(T) }`, or
- a type that implements `std/async::Future` with `Output` and `poll` compatible with Poll-like return.

Otherwise type checking rejects the await expression.

## Lowering Model (User-Visible)

- compiler generates poll/state transitions for async bodies.
- pending branches preserve continuation state.
- completion returns `Poll::Ready(output)`-equivalent behavior at runtime layer.
- await operands must not capture non-static borrows across suspension points.

## Cancellation Hooks

When compiler-generated async entry/test wrappers observe cancellation on `Pending`,
they route through optional `std/async` hooks with this precedence:

1. drop/rebind:
   - `cancel_drop_hint_with(rt, cx, hint, f)` / `cancel_drop_hint(cx, hint, f)`
   - then `cancel_drop_state_with` / `cancel_drop_with` / `cancel_drop_state` / `cancel_drop`
2. cleanup:
   - `cancel_cleanup_hint_with(rt, cx, hint)` / `cancel_cleanup_hint(cx, hint)`
   - then `cancel_cleanup_state_with` / `cancel_cleanup_with` / `cancel_cleanup_state` / `cancel_cleanup`
3. return:
   - `cancel_return_hint_with(rt, cx, hint)` / `cancel_return_hint(cx, hint)`
   - then `cancel_return_with` / `cancel_return`
   - else fallback default return value for the function return type.

`hint` is produced from `cancel_hint(cx, frame_state, spins)` when available.
Default `std/async::CancelHint` also carries coarse reclaim intent (`reclaim`):
- `cancel_reclaim_keep()`
- `cancel_reclaim_shallow()`
- `cancel_reclaim_deep()`
computed by `cancel_reclaim_from_state_spins(state, spins)`.

## Diagnostics

Parser errors:

- malformed await syntax (for example missing receiver)

Type/lowering errors:

- `await can only be used in async fn`
- `` `await` requires Poll-like enum { Pending, Ready(T) } or std/async::Future impl ``
- `await operand cannot contain non-static borrow`
- `await in unsupported statement position`
- `impl must use async fn to implement async trait method`
- `impl async method output type mismatch: got <T0>, want <T1>`

## Example

```vox
trait Work { async fn run(x: Self) -> i32; }
struct F { v: i32 }
impl Work for F { async fn run(x: F) -> i32 { return x.v; } }

async fn get() -> i32 { return 1; }

async fn main() -> i32 {
  let x: i32 = get().await;
  let y: i32 = if x > 0 { get().await } else { 0 };
  let z: i32 = match y { 0 => 0, _ => get().await };
  return z + Work.run(F { v: 2 }).await;
}
```
