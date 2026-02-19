# Literals and Default Typing

## Scope

Defines literal forms and typing behavior used by the type checker.

Coverage IDs: `S002`, `S005`, `S018`, `S021`, `S022`.

## Integer Literals

- Decimal integer literals are supported.
- Literal type is inferred from context when possible.
- Out-of-domain literal usage for target type is rejected.

## Float Literals

- Decimal floating literals are supported.
- Type is resolved from context (`f32`/`f64`) or default rules where applicable.
- Exponent and suffixed forms are supported (for example `1e3`, `2.5e-2f32`).

## Boolean Literals

- `true`
- `false`

## Character Literals

- Single character in single quotes, for example `'A'`.
- Escape forms are supported per lexer/parser implementation.

## String Literals

- Double-quoted string literals, for example `"vox"`.
- Triple-quoted multiline text literals are supported and normalized by indentation rules.

### Triple-Quoted Multiline Rules

For `"""..."""` literals, parser normalization is:

1. line ending normalization (`\r\n`/`\r` -> `\n`);
2. if the first body byte is `\n`, it is removed;
3. one trailing empty line is removed;
4. common leading-space indentation across non-empty lines is removed;
5. tab indentation is rejected with parse error;
6. escape decoding is then applied on normalized text.

Example source:

```vox
let s: String = """
    line1
      line2
    line3
""";
```

Normalized runtime text is:

```text
line1
  line2
line3
```

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
