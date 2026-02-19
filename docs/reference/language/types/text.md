# Text Types

## Scope

Defines owned and borrowed text types: `String`, `&str`, and `&'static str`.

## Grammar

```vox
Type
  := "String"
   | "str"
   | "&str"
   | "&'static str"
```

Parser accepts `str` in type positions, but type checking rejects bare `str` values.

## `String`

- Owned string type.
- Usable as local values, struct fields, and return values.
- Common operations come from std/prelude methods (for example `len`, `concat`).

## Borrowed Text (`&str`, `&'static str`)

- `&str` is a non-static borrowed text view.
- `&'static str` is a static borrowed text view.
- String literals can be used where borrowed text is expected.

## Bare `str` Rule

Bare `str` is rejected by the type checker:

```text
bare str is not allowed; use String for owned text or &str for borrowed text
```

Use `String` for owned text, and `&str`/`&'static str` for borrowed text.

## Interop Notes

- String literals can type-check as borrowed string forms where expected.
- Explicit conversion APIs in std may be used when ownership changes are required.

## Type Errors

Type checking rejects:

- bare `str` in value/parameter/return positions,
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
