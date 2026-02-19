# Primitive Types

## Scope

Defines built-in scalar types and their core semantics.

## Grammar

```vox
Type
  := "bool"
   | "char"
   | IntType
   | FloatType

IntType
  := "i8" | "i16" | "i32" | "i64" | "isize"
   | "u8" | "u16" | "u32" | "u64" | "usize"

FloatType
  := "f32" | "f64"
```

## Integer Types

Signed integers:

- `i8`, `i16`, `i32`, `i64`, `isize`

Unsigned integers:

- `u8`, `u16`, `u32`, `u64`, `usize`

Notes:

- `isize`/`usize` are pointer-width integers.
- Integer arithmetic and comparisons are supported.
- Bitwise and shift operators are defined for integer types.

## Floating Types

- `f32`
- `f64`

Notes:

- Floating-point arithmetic and comparisons are supported.
- Integer range refinement (`@range`) does not apply to float base types.

## Boolean Type

- `bool` has two values: `true`, `false`.
- Logical operators (`&&`, `||`, `!`) require boolean operands.

## Character Type

- `char` represents a Unicode scalar value.
- Current runtime representation is `u32` codepoint-compatible.

## Type Errors

Type checking rejects:

- invalid operator/type combinations (for example `true + 1`),
- assignments where source and target types are incompatible,
- implicit casts that require explicit `as`.

## Example

```vox
fn classify(x: i32, y: f64, ok: bool, ch: char) -> i32 {
  let a: i64 = x as i64;
  let b: f64 = y * 2.0;
  if ok && (ch as i32) > 64 {
    return (a as i32) + (b as i32);
  }
  return 0;
}
```
