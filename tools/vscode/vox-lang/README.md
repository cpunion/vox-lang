# Vox VSCode Extension

This extension starts `vox lsp` for `.vox` files and provides:
- diagnostics
- document formatting
- basic syntax highlighting

## Local development

1. Build or prepare a `vox` executable in your `PATH` (or set `vox.compilerPath`).
2. Install extension deps:
   ```bash
   cd tools/vscode/vox-lang
   npm install
   ```
3. Open this folder in VSCode and press `F5` to launch an Extension Development Host.

## Settings

- `vox.compilerPath`: path to `vox` compiler executable (`vox` by default)
- `vox.nodePath`: node executable path for `vox lsp` (`node` by default)
