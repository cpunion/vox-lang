# While

Covered form:

- `while` loop

Example:

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
