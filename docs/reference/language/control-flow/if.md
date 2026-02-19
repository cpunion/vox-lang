# If

## Scope

Defines `if` statement and `if` expression forms.

Coverage IDs: `S101`, `S102`.

## Grammar (Simplified)

```vox
IfStmt
  := "if" Expr Block ("else" IfStmt | "else" Block)?

IfExpr
  := "if" Expr Block "else" Block
```

## Statement Form

```vox
if cond { ... }
if cond { ... } else { ... }
if cond { ... } else if cond2 { ... } else { ... }
```

## Expression Form

```vox
let v: T = if cond { expr1 } else { expr2 };
```

- expression form requires `else` branch.
- then/else branch results must be type-compatible.

## Semantics

- condition expression is evaluated first.
- condition must type-check as `bool`.
- exactly one branch executes.

## Diagnostics

Parser errors:

- missing/malformed blocks or `else` structure

Type errors:

- non-boolean condition
- incompatible branch result types in expression context

## Examples

```vox
fn classify(x: i32) -> i32 {
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
