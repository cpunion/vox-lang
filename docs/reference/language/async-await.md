# Async and Await

## Scope

Coverage IDs: `S501`, `S502`, `S503`, `S504`, `S505`.

## Syntax

Async function:

```vox
async fn name(args...) -> Ret { ... }
```

Await expression:

```vox
expr.await // recommended
await expr // compatibility form
```

Async trait method:

```vox
trait T {
  async fn m(x: Self) -> Ret;
}
```

## Semantics

- `async fn` returns a future-like value lowered by the compiler.
- `.await` polls the awaited future and yields the ready value.
- `await` is supported inside expression boundaries such as `if` and `match` arms.

## Diagnostics

- malformed await expressions (for example `.await` without receiver) are parse errors.
- invalid await placement or future/type mismatches are type/lowering errors.

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
