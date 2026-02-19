# While

## Scope

Coverage ID: `S103`.

## Syntax

```vox
while <cond-expr> { <stmts> }
```

## Semantics

- Condition is checked before each iteration.
- Loop exits when condition becomes false.
- `break` and `continue` can be used inside loop body.

## Diagnostics

- Malformed condition/body produces parse errors.
- Condition type mismatch is reported by type checking.

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
