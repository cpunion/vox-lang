# Enum Types

## Scope

Defines `enum` declaration syntax, variant constructors, and pattern matching use.

Coverage IDs: `S007`, `S009`, `S010`, `S020`.

## Grammar (Simplified)

```vox
EnumDecl
  := Vis? "enum" Ident GenericParams? WhereClause? "{" VariantList? "}"

Variant
  := Ident ("(" TypeList? ")")?

VariantCtorExpr
  := TypePath "." Variant "(" ArgList? ")"
   | TypePath "." Variant
   | "." Variant "(" ArgList? ")"
   | "." Variant

VariantPattern
  := TypePath "." Variant "(" PatList? ")"
   | TypePath "." Variant
   | "." Variant "(" PatList? ")"
   | "." Variant
```

Current declaration coverage supports tuple-style payload variants and unit variants.

## Declaration

- Enum defines a nominal sum type with named variants.
- Variant payloads are positional (tuple-style).
- Type parameters and `where` constraints are supported.

## Construction and Match

- Constructor call form: `Option[i32].Some(1)`.
- Unit variant form: `Option[i32].None`.
- Contextual dot-shorthand is accepted in typed contexts:
  - constructor: `.Some(1)`, `.None`
  - pattern: `.Some(x)`, `.None`
- Match patterns use qualified variant names (for example `Option.Some(x)`).
  Generic arguments are typically inferred from the matched value and can be omitted.

## Diagnostics

Parser errors:

- malformed variant list or payload type list
- malformed variant constructor/pattern syntax

Type/check errors:

- variant not found on enum type
- payload arity/type mismatch
- incompatible match arm result types

## Example

```vox
enum Option[T] {
  Some(T),
  None,
}

fn main() -> i32 {
  let o: Option[i32] = Option[i32].Some(3);
  return match o {
    Option.Some(x) => x,
    Option.None => 0,
  };
}
```
