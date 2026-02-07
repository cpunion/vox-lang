package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStage1ToolchainBuildsMultiModuleProgram(t *testing.T) {
	// 1) Build the stage1 compiler (vox_stage1) using stage0.
	stage1Dir := filepath.Clean(filepath.Join("..", "..", "..", "stage1"))
	stage1Bin, err := compile(stage1Dir)
	if err != nil {
		t.Fatalf("build stage1 failed: %v", err)
	}

	// 2) Create a tiny multi-module program.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src", "a"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Dependency package.
	depRoot := filepath.Join(root, "dep_pkg")
	if err := os.MkdirAll(filepath.Join(depRoot, "src"), 0o755); err != nil {
		t.Fatalf("mkdir dep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "Vox.toml"), []byte(`[package]
name = "dep"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write dep Vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "src", "dep.vox"), []byte("pub fn two() -> i32 { return 2; }\n"), 0o644); err != nil {
		t.Fatalf("write dep src: %v", err)
	}

	// Root package manifest (uses dep by path).
	if err := os.WriteFile(filepath.Join(root, "Vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "dep_pkg" }
`), 0o644); err != nil {
		t.Fatalf("write Vox.toml: %v", err)
	}

	mainSrc := "import \"std/prelude\" as prelude\nimport \"a\" as a\nimport \"dep\" as dep\nfn main() -> i32 { prelude.assert(true); return a.one() + dep.two(); }\n"
	aSrc := "pub fn one() -> i32 { return 1; }\n"
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte(mainSrc), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "a", "a.vox"), []byte(aSrc), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}

	// 3) Use stage1 compiler to build it (auto-discover src/**.vox).
	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("stage1 build failed: %v", err)
	}

	// 4) Run the produced binary and check output (driver prints main return).
	run := exec.Command(outBin)
	run.Dir = root
	b, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run built program failed: %v", err)
	}
	if got := strings.TrimSpace(string(b)); got != "3" {
		t.Fatalf("unexpected output: %q", got)
	}
}
