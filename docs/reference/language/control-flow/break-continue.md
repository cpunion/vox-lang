# Break and Continue

## Scope

Coverage ID: `S105` (currently partial: `loop` keyword coverage is pending).

## Syntax

```vox
break;
continue;
```

## Semantics

- `break` exits the nearest enclosing loop.
- `continue` skips the rest of current iteration and starts next iteration.

## Diagnostics

- Using `break` / `continue` outside loops is rejected during type checking/lowering.

## Example

```vox
fn main() -> i32 {
  let mut i: i32 = 0;
  let mut s: i32 = 0;
  while i < 3 {
    i = i + 1;
    if i == 2 {
      continue;
    } else {
      s = s + i;
    }
  }
  while true {
    s = s + 1;
    break;
  }
  return s;
}
```
