# Break and Continue

Covered forms:

- `break`
- `continue`

Example:

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
