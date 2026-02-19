# Functions and Calls

## Scope

Defines function declaration, parameter/return typing, call forms, and block expression rules.

Coverage IDs: `S201`, `S202`, `S203`, `S204`, `S205`.

## Grammar (Simplified)

```vox
FnDecl
  := "fn" Ident GenericParams? "(" ParamList? ")" "->" Type Block

Param
  := Ident ":" Type

CallExpr
  := Expr "(" ArgList? ")"

MemberCallExpr
  := Expr "." Ident "(" ArgList? ")"

UfcsCallExpr
  := TypeOrTrait "." Ident "(" ArgList? ")"

BlockExpr
  := "{" Stmt* TailExpr? "}"
```

## Function Declarations

- A function has a name, parameter list, explicit return type, and body block.
- Unit return uses `-> ()`.
- Generics and `where` constraints are allowed where supported by parser/typechecker.

## Calls

### Direct/Member Call

- `f(x, y)` calls callable expression `f`.
- `recv.method(args...)` resolves via inherent impl or trait method resolution.
- method resolution in current checker prefers:
  - built-in receiver methods (`Vec`, `String`, primitive helpers),
  - bound-trait methods for type parameters,
  - inherent impl methods,
  - trait dispatch methods

### UFCS Call

- `Trait.method(recv, args...)` calls a method using explicit trait/type qualification.
- `alias.Trait.method(...)` and imported named trait forms are supported by the same path logic.

## Borrowed Parameter Arguments

- For parameters typed as `&T`, call arguments must be place expressions.
- For parameters typed as `&mut T`, call arguments must be mutable place expressions.
- Non-place or immutable-place arguments are rejected at call checking time.

## Special Constructor Form

- `Vec()` requires expected type context (`Vec[T]`); otherwise type checking fails.

## Return Semantics

- `return expr;` exits current function immediately.
- Returned value must type-check against declared return type.

## Block Expression

- A block used in expression position evaluates to its tail expression.
- If no tail expression, block result is `()`.

## Diagnostics

Parser errors:

- malformed function signature/body
- malformed parameter list or call argument list

Type errors:

- argument count/type mismatch
- return type mismatch
- unresolved method/UFCS targets
- representative diagnostics include:
  - `arg #N type mismatch: expected ...`
  - `call arg #N for & parameter must be place`
  - `call arg #N for &mut parameter must be mutable place`
  - `call arg #N for &mut parameter is immutable`
  - `ambiguous trait method call: ...`
  - `unknown trait method: ...`

## Example

```vox
struct I { v: i32 }
impl I {
  fn inc(self: Self) -> i32 { return self.v + 1; }
}
trait Add { fn add(self: Self, y: i32) -> i32; }
impl Add for I { fn add(self: Self, y: i32) -> i32 { return self.v + y; } }

fn add1(x: i32) -> i32 {
  let y: i32 = { let z: i32 = x + 1; z };
  return y;
}

fn main() -> i32 {
  let i: I = I { v: 2 };
  return add1(i.inc()) + Add.add(i, 3);
}
```
