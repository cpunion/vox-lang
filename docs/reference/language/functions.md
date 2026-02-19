# Functions and Calls

## Scope

Coverage IDs: `S201`, `S202`, `S203`, `S204`, `S205`.

## Syntax

Function declaration:

```vox
fn <name>(<params>) -> <ret-ty> { <stmts> }
```

Call forms:

- instance member call: `recv.method(args...)`
- UFCS call: `Trait.method(recv, args...)`

Block expression:

```vox
let x: T = { <stmts>; <tail-expr> };
```

## Semantics

- `return` exits current function.
- Method call syntax resolves to impl/trait methods.
- Block expressions evaluate to their tail expression value.

## Diagnostics

- Missing function body is a parse error.
- Signature/body type mismatch is a type error.

## Example

```vox
struct I { v: i32 }
impl I {
  fn inc(self: Self) -> i32 { return self.v + 1; }
}
trait Add { fn add(self: Self, y: i32) -> i32; }
impl Add for I { fn add(self: Self, y: i32) -> i32 { return self.v + y; } }

fn add1(x: i32) -> i32 {
  let y: i32 = { let z: i32 = x + 1; z };
  return y;
}

fn main() -> i32 {
  let i: I = I { v: 2 };
  return add1(i.inc()) + Add.add(i, 3);
}
```
