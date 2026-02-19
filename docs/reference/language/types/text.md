# Text Types

## Scope

Defines `String` and `str` related typing rules.

## Grammar

```vox
Type
  := "String"
   | "str"
   | "&str"
   | "&'static str"
```

`str` is dynamically-sized and is normally used through references.

## `String`

- Owned string type.
- Usable as local values, struct fields, and return values.
- Common operations come from std/prelude methods (for example `len`, `concat`).

## `str`

- Non-owned string view target type.
- Typically appears as `&str` or `&'static str`.
- Plain `str` value positions are not generally used as standalone sized values.

## Interop Notes

- String literals can type-check as borrowed string forms where expected.
- Explicit conversion APIs in std may be used when ownership changes are required.

## Type Errors

Type checking rejects:

- mismatched owned vs borrowed string forms without valid conversion,
- invalid mutation through immutable string borrows.

## Example

```vox
fn greet(name: &str) -> String {
  let p: String = "hello, ";
  return p.concat(name.to_string());
}

fn banner() -> &'static str {
  return "vox";
}
```
