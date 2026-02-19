# Error Handling Expressions

## Scope

Defines parser-level syntax for:

- question propagation operator `expr?`,
- try block expression `try { ... }`.

Coverage IDs: `S108`, `S109`, `S110`.

## Grammar (Simplified)

```vox
QuestionExpr
  := Expr "?"

TryBlockExpr
  := "try" BlockExpr
```

## Forms

### Question Operator

```vox
let v: i32 = get()?;
```

`?` is a postfix operator applied to an expression.

### Try Block

```vox
let r: Result[i32, String] = try {
  let x: i32 = get()?;
  x + 1
};
```

`try { ... }` is an expression form and can appear anywhere an expression is expected.

## Diagnostics

Parser errors include:

- malformed `try` form without block body,
- malformed question syntax in invalid positions.

Type-level propagation constraints are specified by the checker/runtime model
and are validated after parsing.
