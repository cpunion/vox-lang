# Vox Syntax Reference (Current)

This document reflects the syntax surface currently covered by merged acceptance tests.

## 1. Basic Types and Literals

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

## 2. Control Flow

- `if/else` statement
- `if` expression
- `while` loop
- `break` and `continue`
- `match` expression

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
  let v: i32 = if s > 0 { 1 } else { 0 };
  let m: i32 = match v { 1 => 7, _ => 0 };
  return m;
}
```

## 3. Operators and Cast

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

## 4. Functions, Method Call, UFCS

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

## 5. Generics

- generic type parameters and explicit type arguments
- const generics
- `where` trait bounds
- `where comptime` bounds
- impl-head `where comptime`
- type pack / variadic parameter surface syntax (`T...`)

Example:

```vox
fn id[T](x: T) -> T where T: Eq { return x; }
fn addn[const N: i32](x: i32) -> i32 where comptime N > 0 { return x + N; }
fn sum[T](head: T, tail: T...) -> T { return head; }
trait Tag { fn tag(x: Self) -> i32; }
impl[T] Tag for Vec[T] where comptime @size_of(T) <= 16 { fn tag(x: Vec[T]) -> i32 { return 1; } }
```

## 6. Traits and Impls

- trait declaration
- trait default method body
- associated type in trait/impl
- supertrait syntax (`trait A: B`)
- generic impl head (`impl[T] ...`)
- negative impl syntax (`impl !Trait for Type {}`)

Example:

```vox
trait Base { fn base(x: Self) -> i32; }
trait Iter: Base {
  type Item;
  fn next(x: Self) -> Self.Item;
  fn ready(x: Self) -> bool { return true; }
}
struct I { v: i32 }
impl Base for I { fn base(x: I) -> i32 { return x.v; } }
impl Iter for I {
  type Item = i32;
  fn next(x: I) -> i32 { return x.v; }
}
impl !Send for I {}
```

For precise status by syntax ID, see `docs/reference/syntax-coverage.md`.
