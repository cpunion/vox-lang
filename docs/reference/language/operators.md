# Operators and Cast

Current covered operators:

- arithmetic: `+ - * / %`
- logical: `&& || !`
- bitwise/shift: `& | ^ << >>`
- comparison/equality: `< <= > >= == !=`
- cast: `as`

Example:

```vox
fn main() -> i32 {
  let a: i32 = 5;
  let b: i32 = 2;
  let c: bool = (a > b) && (a != 0) || !(b == 3);
  let d: i32 = (a + b) * (a - b) / b % 3;
  let e: i32 = a << 1 | b ^ a & b;
  let f: i64 = a as i64;
  if c && f > 0 as i64 { return d + e; }
  return 0;
}
```
