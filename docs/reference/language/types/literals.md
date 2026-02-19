# Literals and Default Typing

## Scope

Defines literal forms and typing behavior used by the type checker.

Coverage IDs: `S002`, `S005`.

## Integer Literals

- Decimal integer literals are supported.
- Literal type is inferred from context when possible.
- Out-of-domain literal usage for target type is rejected.

## Float Literals

- Decimal floating literals are supported.
- Type is resolved from context (`f32`/`f64`) or default rules where applicable.

## Boolean Literals

- `true`
- `false`

## Character Literals

- Single character in single quotes, for example `'A'`.
- Escape forms are supported per lexer/parser implementation.

## String Literals

- Double-quoted string literals, for example `"vox"`.
- Triple-quoted multiline text literals are supported and normalized by indentation rules.

## Parse Errors

Malformed literals fail during parse/lex stage, for example:

- unterminated string/char literal,
- invalid literal syntax.

## Example

```vox
fn lit() -> i32 {
  let a: i32 = 42;
  let b: f64 = 3.14;
  let c: bool = true;
  let d: char = 'V';
  let s: String = "vox";
  return a + (b as i32) + (d as i32) + s.len() + if c { 1 } else { 0 };
}
```
