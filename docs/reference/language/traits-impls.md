# Traits and Impls

## Scope

Defines trait declarations, associated items, impl forms, inherent impls, and negative impl syntax.

Coverage IDs: `S401`, `S402`, `S403`, `S404`, `S405`, `S406`, `S407`.

## Grammar (Simplified)

```vox
TraitDecl
  := "trait" Ident SuperTraitClause? "{" TraitItem* "}"

SuperTraitClause
  := ":" TraitPath ("+" TraitPath)*

TraitItem
  := AssocTypeDecl
   | TraitMethodDecl

AssocTypeDecl
  := "type" Ident ";"

TraitMethodDecl
  := "fn" Ident Signature (";" | Block)

ImplDecl
  := "impl" GenericParams? TraitPath "for" Type WhereClause? "{" ImplItem* "}"

InherentImplDecl
  := "impl" GenericParams? Type WhereClause? "{" ImplItem* "}"

NegativeImplDecl
  := "impl" "!" TraitPath "for" Type "{}"
```

## Trait Declarations

- Traits define required method signatures and optional default bodies.
- Traits may declare associated types.
- Traits may inherit supertraits (`trait A: B`).

## Impl Forms

### Trait Impl

`impl Trait for Type { ... }` provides concrete method/type bindings.

### Generic Trait Impl

`impl[T] Trait for Type[T] { ... }` is supported with optional `where` constraints.

### Inherent Impl

`impl Type { ... }` attaches methods directly to a concrete type (non-trait methods).

### Negative Impl

`impl !Trait for Type {}` declares explicit non-implementation for trait constraints.

## Resolution Notes

- Method lookup may resolve via inherent impl first, then trait context, according to checker rules.
- UFCS form can disambiguate trait method selection.

## Diagnostics

Parser errors:

- malformed trait/impl headers and items
- invalid negative impl grammar forms

Type/check errors:

- trait method signature mismatch in impl
- missing required associated type/method bindings
- invalid overlapping/conflicting impls under checker rules

## Example

```vox
trait Base { fn base(x: Self) -> i32; }
trait Iter: Base {
  type Item;
  fn next(x: Self) -> Self.Item;
  fn ready(x: Self) -> bool { return true; }
}

struct I { v: i32 }
impl I {
  fn inc(self: Self) -> i32 { return self.v + 1; }
}
impl Base for I { fn base(x: I) -> i32 { return x.v; } }
impl Iter for I {
  type Item = i32;
  fn next(x: I) -> i32 { return x.v; }
}
impl !Send for I {}
```
