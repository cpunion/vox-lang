# Break and Continue

## Scope

Defines loop-control statements `break` and `continue`.

Coverage ID: `S105` (partial: dedicated `loop` keyword coverage remains pending).

## Grammar (Simplified)

```vox
BreakStmt    := "break" ";"
ContinueStmt := "continue" ";"
```

## Semantics

- `break` exits nearest enclosing loop.
- `continue` skips to next iteration of nearest enclosing loop.

## Context Rules

- both statements are valid only inside loop bodies.
- control target is lexical nearest loop construct.

## Diagnostics

Type/lowering errors:

- using `break` or `continue` outside loops

Parser errors:

- malformed statement termination

## Example

```vox
fn main() -> i32 {
  let mut i: i32 = 0;
  let mut s: i32 = 0;
  while i < 3 {
    i = i + 1;
    if i == 2 {
      continue;
    }
    s = s + i;
  }
  while true {
    s = s + 1;
    break;
  }
  return s;
}
```
