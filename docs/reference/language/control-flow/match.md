# Match

Covered form:

- `match` expression

Example:

```vox
fn map_flag(v: i32) -> i32 {
  let m: i32 = match v { 1 => 7, _ => 0 };
  return m;
}
```
