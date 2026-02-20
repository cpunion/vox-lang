# Type Conversion and Cast

## Scope

Defines explicit conversion syntax and validation behavior.

Related operator coverage: `S805`.

## Grammar

```vox
CastExpr
  := Expr "as" Type
```

## Rule Summary

- Vox uses explicit cast syntax `as` for cross-type conversion.
- Implicit widening/narrowing conversions are limited; when conversion is required, use `as`.
- Conversion validity is checked by type checker.

## Integer Conversions

- Integer-to-integer casts are supported.
- Narrowing behavior follows runtime/backend conversion semantics.
- Additional runtime checks apply when casting into `@range(...)` refined types.
- Additional runtime checks apply when casting into `@verified(...)` refined types.

## Float and Integer Conversions

- Numeric casts between integer and float types are supported where defined.
- Invalid cast targets are rejected by type checker.

Current checker conversion surface:

- int-like <-> int-like
- float <-> float
- int-like <-> float

## Reference-Related Conversions

- Reference form changes are not general-purpose casts.
- Borrow mutability and lifetime markers must satisfy borrow rules.

## Diagnostics

Type checking rejects:

- casts to unsupported target kinds,
- invalid reference or mutability casts,
- refinement casts violating range constraints (const-time failure or runtime panic).
- refinement casts violating verified predicate constraints (runtime panic).

## Example

```vox
type Byte = @range(0..=255) u16

fn conv(a: i32, b: f64) -> i32 {
  let x: i64 = a as i64;
  let y: i32 = b as i32;
  let z: Byte = a as Byte;
  return (x as i32) + y + (z as i32);
}
```
