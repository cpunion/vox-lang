# Types

Current covered forms:

- integer and bool types
- char and string literals
- borrow/reference forms (`&T`, `&'static str`)
- range-annotated integer type (`@range(lo..=hi)`)

Example:

```vox
fn f(a: i32, b: bool, c: char, d: String, e: &i32, s: &'static str) -> i32 {
  let x: @range(0..=3) i8 = 2 as @range(0..=3) i8;
  if b { return a + (x as i32); }
  return c as i32 + d.len() + e.to_string().len() + s.len();
}
```
