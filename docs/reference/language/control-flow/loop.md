# Loop

## Scope

Defines the unconditional loop form `loop { ... }`.

Coverage ID: `S105` (loop form).

## Grammar (Simplified)

```vox
LoopStmt
  := "loop" Block
```

## Semantics

- `loop` always enters the body.
- termination is explicit via `break` (or function return/panic).
- `continue` jumps to the next iteration.

## Lowering Note

- parser lowers `loop { body }` to `while true { body }`.
- this keeps one canonical loop statement shape in later phases.

## Diagnostics

Parser errors:

- malformed loop body block

Type/check errors:

- invalid `break`/`continue` usage context

## Example

```vox
fn main() -> i32 {
  let mut x: i32 = 0;
  loop {
    x = x + 1;
    if x > 3 {
      break;
    }
  }
  return x;
}
```
