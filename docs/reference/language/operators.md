# Operators and Cast

## Scope

Defines expression operators, precedence, operand requirements, and cast behavior.

Coverage IDs: `S801`, `S802`, `S803`, `S804`, `S805`, `S806`.

## Grammar (Simplified)

```vox
Expr
  := UnaryExpr
   | Expr BinaryOp Expr
   | Expr "as" Type

UnaryExpr
  := "!" Expr
   | "+" Expr
   | "-" Expr

BinaryOp
  := "*" | "/" | "%"
   | "+" | "-"
   | "<<" | ">>"
   | "&" | "^" | "|"
   | "<" | "<=" | ">" | ">=" | "==" | "!="
   | "&&" | "||"
```

## Precedence and Associativity

From high to low:

1. postfix/member/call/await
2. unary (`!`, unary `+`, unary `-`)
3. multiplicative (`*`, `/`, `%`)
4. additive (`+`, `-`)
5. shift (`<<`, `>>`)
6. bitwise (`&`, `^`, `|`)
7. comparison/equality (`< <= > >= == !=`)
8. logical (`&&`, `||`)
Cast note:

- `as` is parsed as a postfix cast on an already-built expression node.
- chaining and interaction with other postfix constructs follow parser rules.

Binary operators are left-associative unless parser/type rules specify otherwise.

## Operator Families

### Arithmetic

- `+ - * / %`
- valid on numeric operands under type-check rules
- both operands must resolve to the same base numeric type (after `@range` base stripping),
  except numeric literals may be context-typed during checking

### Logical

- `&& || !`
- `&&`/`||` operands/results are `bool`
- `!` on `bool` is logical-not
- `&&` and `||` are short-circuit operators

### Bitwise and Shift

- `& | ^ << >>`
- `<< >>` require integer operands
- `& | ^` support integer operands; `bool & bool`, `bool | bool`, `bool ^ bool` are also valid
- unary `!` on integers is bitwise-not

### Comparison and Equality

- `< <= > >= == !=`
- result type is `bool`
- ordered compare (`< <= > >=`) supports:
  - numeric base types and `String`,
  - generic type parameters only when bounded by `Ord`
- equality (`== !=`) supports:
  - primitive scalar types and `String`,
  - generic type parameters only when bounded by `Eq`,
  - struct/enum recursively when all fields are equality-capable

### Cast

- `expr as Type`
- explicit conversion only; implicit conversion is limited
- numeric casts supported in current implementation:
  - int-like <-> int-like
  - float <-> float
  - int-like <-> float
- range-refined target types (`@range(...)`) may insert runtime checks

## Evaluation Semantics

- Expression evaluation follows precedence and associativity.
- `&&` and `||` evaluate right operand conditionally.
- Cast is applied after source expression is evaluated.

## Diagnostics

Parser errors:

- malformed operator expressions (for example `1 + ;`)
- malformed cast syntax

Type errors:

- operand/operator type incompatibility
- unsupported cast target/source pair
- refinement cast violation (const-time error or runtime panic)
- representative diagnostics include:
  - `binary lhs typecheck failed`
  - `binary rhs typecheck failed`

## Example

```vox
fn main() -> i32 {
  let a: i32 = 5;
  let b: i32 = 2;
  let c: bool = (a > b) && (a != 0) || !(b == 3);
  let d: i32 = (a + b) * (a - b) / b % 3;
  let e: i32 = a << 1 | b ^ a & b;
  let f: i64 = a as i64;
  if c && f > 0 as i64 { return d + e; }
  return 0;
}
```
