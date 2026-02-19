# Match

## Scope

Coverage ID: `S106`.

## Syntax

```vox
match <scrutinee-expr> {
  <pattern> => <expr>,
  ...
}
```

Currently covered pattern subset in syntax acceptance:

- literal pattern (for example `1`)
- wildcard pattern (`_`)

## Semantics

- Scrutinee is evaluated once.
- First matching arm is selected.
- In expression position, all arm expression types must be compatible.

## Diagnostics

- Missing arrows/commas/braces produce parse errors.
- Incompatible arm result types produce type errors.

## Example

```vox
fn map_flag(v: i32) -> i32 {
  let m: i32 = match v { 1 => 7, _ => 0 };
  return m;
}
```
