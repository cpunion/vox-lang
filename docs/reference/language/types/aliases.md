# Type Aliases and Labeled Union Aliases

## Scope

Defines `type` declaration forms currently supported:

- simple type alias
- labeled union alias (lowered to enum-like ADT)

Coverage IDs: `S011`, `S013`, `S014`.

## Grammar (Simplified)

```vox
TypeAliasDecl
  := "type" Ident "=" Type

LabeledUnionAliasDecl
  := "type" Ident "=" UnionArm ("|" UnionArm)+

UnionArm
  := Ident ":" Type
```

## Simple Type Alias

```vox
type Meter = i32
```

- introduces a named alias for an existing type expression.
- alias can be used in annotations, params, return types, and item signatures.

## Labeled Union Alias

```vox
type Value = I32: i32 | Str: String
```

Current behavior:

- parser lowers this declaration into an enum-like structure with tagged variants.
  Example:
  ```vox
  type Value = I32: i32 | Str: String
  ```
  is conceptually equivalent to:
  ```vox
  enum Value {
    I32(i32),
    Str(String),
  }
  ```
- each arm label becomes a variant name.
- each arm payload is currently a single positional field type.

Construction/match style is enum-like (for example `Value.I32(7)`, `.I32(v)`).

## Diagnostics

Parser errors:

- missing alias RHS type
- malformed labeled union arm syntax
- malformed `|` chain in union alias

Type/check errors:

- invalid variant constructor/pattern usage against lowered union type

## Example

```vox
type Meter = i32

type Value = I32: i32 | Str: String

fn read(v: Value) -> i32 {
  return match v {
    .I32(x) => x,
    .Str(_s) => 0,
  };
}
```
