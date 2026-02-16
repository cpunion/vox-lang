# Vox LSP Server

`tools/lsp/server.mjs` is a lightweight JSON-RPC language server for Vox.

Current capabilities:
- publish diagnostics on open/change
- full-document formatting

Run manually:

```bash
vox lsp
```

Environment:
- `VOX_BIN`: compiler executable path used by diagnostics (`vox` by default)
