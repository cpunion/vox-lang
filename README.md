# Vox Language

This repository now keeps a single active compiler line (stage2-only).

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

Recommended gates:

```bash
make test-active
make test
```

Quick selfhost smoke:

```bash
./scripts/ci/stage2-rolling-selfhost.sh test
```
