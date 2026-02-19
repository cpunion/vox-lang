# While

## Scope

Defines `while` loop syntax and behavior.

Coverage ID: `S103`.

Related: `docs/reference/language/control-flow/loop.md`.

## Grammar (Simplified)

```vox
WhileStmt
  := "while" Expr Block
```

## Semantics

- condition is evaluated before each iteration.
- loop body executes only when condition is `true`.
- `break` exits loop; `continue` starts next iteration.

## Type Rules

- while condition must be `bool`.

## Diagnostics

Parser errors:

- malformed condition/body

Type/check errors:

- non-boolean condition
- invalid `break`/`continue` usage context

## Example

```vox
fn sum3() -> i32 {
  let mut i: i32 = 0;
  let mut s: i32 = 0;
  while i < 3 {
    i = i + 1;
    s = s + i;
  }
  return s;
}
```
