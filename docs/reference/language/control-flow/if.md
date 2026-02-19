# If

Covered forms:

- `if/else` statement
- `if` expression

Example:

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
