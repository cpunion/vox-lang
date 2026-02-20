# Range-Refined Integer Types

## Scope

Defines integer refinement using `@range`.

Coverage ID: `S004`.

## Grammar

```vox
RangeType
  := "@range(" RangeBound "..=" RangeBound ")" IntBaseType

IntBaseType
  := "i8" | "u8" | "i16" | "u16"
   | "i32" | "u32" | "i64" | "u64"
   | "isize" | "usize" | "char"

RangeBound
  := IntConst | CharConst
```

## Semantics

A range-refined type narrows a base integer type to a closed interval `[lo, hi]`.

Example:

```vox
type Tiny = @range(0..=3) i8
```

Values converted into `Tiny` must be within `0..=3`.

For `char` base, bounds may use char literals and are interpreted as Unicode code points.

```vox
type Lower = @range('a'..='z') char
```

## Checks

- Compile-time: constant out-of-range conversions are rejected.
- Runtime: non-constant conversions emit runtime checks; out-of-range triggers panic.

## Non-Goals (Current)

- Float range refinement is not supported.
- General symbolic interval arithmetic in type-level reasoning is not fully modeled.

## Diagnostics

Typical errors:

- invalid base type (non-integer) in `@range(...) Base`.
- invalid range bounds.
- out-of-range conversion for constant values.

## Example

```vox
type Small = @range(1..=5) i32

fn pick(x: i32) -> Small {
  return x as Small;
}
```
