# Functions and Calls

Current covered forms:

- function declaration and return
- instance member call (`x.m(...)`)
- UFCS style call (`Trait.m(x, ...)`)
- block expression

Example:

```vox
struct I { v: i32 }
impl I {
  fn inc(self: Self) -> i32 { return self.v + 1; }
}
trait Add { fn add(self: Self, y: i32) -> i32; }
impl Add for I { fn add(self: Self, y: i32) -> i32 { return self.v + y; } }

fn add1(x: i32) -> i32 {
  let y: i32 = { let z: i32 = x + 1; z };
  return y;
}

fn main() -> i32 {
  let i: I = I { v: 2 };
  return add1(i.inc()) + Add.add(i, 3);
}
```
