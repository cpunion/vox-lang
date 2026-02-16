# Vox Language

This repository keeps a single active compiler line.

- Active compiler source: `src/`
- Active package manifest: `vox.toml`
- Active build/test outputs: `target/`

Historical bootstrap lines (`stage0` / `stage1`) are archived on branch:

- `archive/stage0-stage1`

Start here:

- `docs/00-overview.md`
- `docs/README.md`
- `docs/15-toolchain.md`
- `docs/24-release-process.md`

Install:

```bash
# download release binary to ~/.vox/bin
curl -fsSL https://raw.githubusercontent.com/cpunion/vox-lang/main/install.sh | bash

# local rolling-selfhost build install (run inside repo)
bash install.sh --local
```

Recommended gates:

```bash
make test-active
make test-public-api
make test
```

Project commands:

```bash
vox build
vox test
vox install
```

Quick selfhost smoke:

```bash
./scripts/ci/rolling-selfhost.sh test
```

Formatter:

```bash
make fmt
make fmt-check
```

Language server:

```bash
vox lsp
```

VSCode extension source:

- `tools/vscode/vox-lang`
