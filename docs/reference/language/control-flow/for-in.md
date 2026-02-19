# For-In

## Scope

Defines `for <name> in <expr> { ... }` loop syntax.

Coverage ID: `S104`.

## Grammar (Simplified)

```vox
ForInStmt
  := "for" Ident "in" Expr Block
```

## Parser Lowering

Current lowering keeps later phases unchanged by rewriting `for-in` to a block with `while`:

```vox
for x in xs { body }
```

lowers to:

```vox
{
  let __vox_for_iter_<line>_<col> = xs;
  let mut __vox_for_idx_<line>_<col> = 0;
  while __vox_for_idx_<line>_<col> < __vox_for_iter_<line>_<col>.len() {
    let x = __vox_for_iter_<line>_<col>.get(__vox_for_idx_<line>_<col>);
    body
    __vox_for_idx_<line>_<col> = __vox_for_idx_<line>_<col> + 1;
  }
}
```

## Semantics

- iteration source is evaluated once.
- iteration index starts at `0` and increments by `1`.
- loop continues while `idx < iter.len()`.
- current item is produced by `iter.get(idx)` and bound to loop variable.
- `break` and `continue` behave as normal loop control.

## Diagnostics

Parser errors:

- missing `in` keyword
- malformed iterable expression or body block

Type/check errors (from lowered form):

- iterable expression type does not support `.len()` / `.get(i32)`
- item binding type mismatch inferred from lowered `let`

## Example

```vox
fn sum(xs: Vec[i32]) -> i32 {
  let mut s: i32 = 0;
  for x in xs {
    s = s + x;
  }
  return s;
}
```
