# Struct Types

## Scope

Defines `struct` declaration syntax, field visibility, generics/where clauses,
struct literals, and field access.

Coverage IDs: `S006`, `S008`, `S010`.

## Grammar (Simplified)

```vox
StructDecl
  := Vis? "struct" Ident GenericParams? WhereClause? "{" FieldList? "}"

Field
  := Vis? Ident ":" Type

StructLit
  := TypePath "{" StructLitFieldList? "}"

StructLitField
  := Ident ":" Expr

FieldAccess
  := Expr "." Ident
```

## Declaration

- Struct defines a nominal product type with named fields.
- Field visibility can be declared via `pub` on each field.
- Type parameters and `where` constraints are supported.

## Struct Literal

- Literal uses named fields (`S { a: 1, b: 2 }`).
- Generic type arguments may be explicit (`Pair[i32] { ... }`).

## Field Access

- `expr.field` reads a field value.
- mutability/borrow checks are enforced by type checker.

## Diagnostics

Parser errors:

- malformed field syntax (missing `:` etc.)
- malformed struct literal field list

Type/check errors:

- unknown field names
- missing required fields / extra fields
- visibility violations across module boundaries

## Example

```vox
pub struct Pair[T] where comptime @size_of(T) <= 16 {
  pub a: T,
  b: T,
}

fn main() -> i32 {
  let p: Pair[i32] = Pair[i32] { a: 1, b: 2 };
  return p.a + p.b;
}
```
