# Macros

## Scope

Defines parser-level macro syntax forms:

- call sugar `name!(...)`,
- builtin `compile!(...)`,
- quote/unquote surface syntax (`quote expr { ... }`, `$x`).

Coverage IDs: `S901`, `S902`, `S903`, `S904`, `S905`.

## Grammar (Simplified)

```vox
MacroCallExpr
  := Expr "!" "(" ArgList? ")"

MacroCallWithTArgs
  := Expr "[" TypeOrConstArgList "]" "!" "(" ArgList? ")"

CompileBangExpr
  := "compile" "!" "(" Expr ")"

QuoteExprSurface
  := "quote" "expr" BlockExpr

DollarUnquote
  := "$" Ident
```

## Forms

### Call Sugar

```vox
id!(1)
addn[3]!(40)
id[i32]!(x)
```

The `!` call form is parsed as macro-call syntax and then handled by macro expansion.

### Builtins

```vox
compile!(1 + 2)
quote expr { $x + 1 }
__file!()
__line!()
__col!()
__module_path!()
__func!()
__caller!()
```

- `compile!(...)` is compile-time splice syntax.
- `quote expr { ... }` is parsed as quote sugar.
- `$x` is parsed as unquote sugar inside quote contexts.
- introspection builtins return caller/module/source metadata at macro expansion stage:
  `__file!`, `__line!`, `__col!`, `__module_path!`, `__func!`, `__caller!`.

## Diagnostics

Parser errors include:

- malformed `!` call form (for example missing `(` after `!`),
- malformed `quote expr { ... }` surface syntax.
