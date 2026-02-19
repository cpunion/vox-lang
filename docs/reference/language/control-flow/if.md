# If

## Scope

Coverage IDs: `S101`, `S102`.

## Syntax

Statement form:

```vox
if <cond-expr> { <stmts> } else { <stmts> }
if <cond-expr> { <stmts> } else if <cond-expr> { <stmts> } else { <stmts> }
```

Expression form:

```vox
let v: T = if <cond-expr> { <expr> } else { <expr> };
```

## Semantics

- `<cond-expr>` is evaluated first.
- When condition is true, then-branch executes; otherwise else-branch executes.
- In expression form, branch result types must be compatible with the expected type.

## Diagnostics

- Missing required blocks or malformed `else` branches produce parse errors.
- Type mismatch between expression branches produces type errors.

## Examples

```vox
fn main() -> i32 {
  let x: i32 = 1;
  if x > 0 {
    return 1;
  } else {
    return 0;
  }
}
```

```vox
fn sign(x: i32) -> i32 {
  let s: i32 = if x > 0 { 1 } else { 0 };
  return s;
}
```
