# Match

## Scope

Defines `match` expression syntax and arm typing behavior.

Coverage ID: `S106`.

## Grammar (Simplified)

```vox
MatchExpr
  := "match" Expr "{" MatchArmList "}"

MatchArm
  := Pattern "=>" Expr ","?
```

## Pattern Subset (Current Acceptance Coverage)

- literal pattern (for example `1`)
- wildcard pattern (`_`)

## Semantics

- scrutinee expression is evaluated once.
- first matching arm is selected.
- `match` in expression position yields selected arm value.

## Type Rules

- arm result types must be compatible in expression context.

## Diagnostics

Parser errors:

- missing/malformed `=>`, commas, or braces

Type/check errors:

- incompatible arm result types
- invalid pattern usage under current parser/type rules

## Example

```vox
fn map_flag(v: i32) -> i32 {
  let m: i32 = match v {
    1 => 7,
    _ => 0,
  };
  return m;
}
```
