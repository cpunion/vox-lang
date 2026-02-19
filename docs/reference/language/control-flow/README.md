# Control Flow

## Scope

Defines statement/expression control-flow forms and malformed control-flow parse coverage.

Coverage IDs: `S101`, `S102`, `S103`, `S104`, `S105`, `S106`, `S107`.

## Documents

- `docs/reference/language/control-flow/if.md`
- `docs/reference/language/control-flow/while.md`
- `docs/reference/language/control-flow/for-in.md`
- `docs/reference/language/control-flow/loop.md`
- `docs/reference/language/control-flow/break-continue.md`
- `docs/reference/language/control-flow/match.md`

## Parse Failure Coverage (`S107`)

Malformed control-flow forms (for example missing block delimiters or invalid control-flow shape)
are tracked by `S107` and covered in:

- `tests/syntax/src/control_flow_test.vox`
