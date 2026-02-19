# Types

## Scope

This page describes currently supported basic type forms.

Coverage IDs: `S001`, `S002`, `S003`, `S004`.

## Built-in Scalar Types

### Integers

Signed: `i8 i16 i32 i64 isize`  
Unsigned: `u8 u16 u32 u64 usize`

### Floating Point

`f32 f64`

### Other Scalars

- `bool`
- `char` (current implementation uses `u32` codepoint representation)

## Text Types

- `String`: owned string type.
- `str`: dynamically-sized string view target; normally used through references (`&str`, `&'static str`).

## Borrow/Reference Types

Supported reference forms:

- shared borrow: `&T`
- mutable borrow: `&mut T`
- static shared borrow: `&'static T`
- static mutable borrow: `&'static mut T`

Examples:

```vox
fn read(x: &i32, s: &'static str) -> i32 { return x.to_string().len() + s.len(); }
fn write(x: &mut i32) -> () { *x = *x + 1; }
```

## Range-Annotated Integer Type

Syntax:

```vox
@range(lo..=hi) T
```

Current constraints:

- `T` must be an integer-family type (`i8/u8/i16/u16/i32/u32/i64/u64/isize/usize`).
- `f32/f64` are not supported as range base types.

Check behavior:

- At compile-time: if conversion value is constant and out-of-range, compilation fails.
- At runtime: when converting into range type from non-constant value, runtime check is inserted; out-of-range panics.

## Literal Notes

- char literals are supported (for example `'A'`, `'\n'`, `'你'`).
- string literals are supported (`"hello"`).

## Example

```vox
fn f(a: i32, b: bool, c: char, d: String, e: &i32, s: &'static str) -> i32 {
  let x: @range(0..=3) i8 = 2 as @range(0..=3) i8;
  if b { return a + (x as i32); }
  return c as i32 + d.len() + e.to_string().len() + s.len();
}
```
