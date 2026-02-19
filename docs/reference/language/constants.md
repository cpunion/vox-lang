# Constants

## Scope

Defines top-level constant item declaration syntax and parse/type boundaries.

Coverage IDs: `S012`, `S014`.

## Grammar (Simplified)

```vox
ConstDecl
  := "const" Ident ":" Type "=" Expr
```

## Semantics

- `const` introduces an immutable named constant item.
- declaration includes explicit type annotation and initializer expression.
- constants can be referenced by later declarations and function bodies according to name resolution rules.

## Diagnostics

Parser errors:

- missing `:` type annotation
- missing initializer expression
- malformed declaration structure

Type/check errors:

- initializer expression type mismatch with declared type
- invalid constant expression forms under const-evaluation rules

## Example

```vox
type Meter = i32
const START: Meter = 1

fn main() -> i32 {
  return START;
}
```
