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
- Await is supported inside expression boundaries currently covered by parser/lowering
  (including `if`/`match` expression contexts covered by acceptance tests).

## Lowering Model (User-Visible)

- compiler generates poll/state transitions for async bodies.
- pending branches preserve continuation state.
- completion returns `Poll::Ready(output)`-equivalent behavior at runtime layer.

## Diagnostics

Parser errors:

- malformed await syntax (for example missing receiver)

Type/lowering errors:

- awaiting non-awaitable/non-future values
- unsupported await placement in contexts not yet lowered
- trait async signature/impl mismatch

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
