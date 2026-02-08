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
	if err := os.WriteFile(filepath.Join(depRoot, "vox.toml"), []byte(`[package]
name = "dep"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write dep vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "src", "dep.vox"), []byte("pub fn two() -> i32 { return 2; }\n"), 0o644); err != nil {
		t.Fatalf("write dep src: %v", err)
	}

	// Root package manifest (uses dep by path).
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "dep_pkg" }
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
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

func TestStage1ToolchainTestPkgRunsTests(t *testing.T) {
	// 1) Build the stage1 compiler (vox_stage1) using stage0.
	stage1Dir := filepath.Clean(filepath.Join("..", "..", "..", "stage1"))
	stage1Bin, err := compile(stage1Dir)
	if err != nil {
		t.Fatalf("build stage1 failed: %v", err)
	}

	// 2) Create a tiny package with:
	// - internal test file under src/**_test.vox (same package, can access private symbols)
	// - integration test under tests/**.vox (external tests module, uses pub API)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src", "a"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tests"), 0o755); err != nil {
		t.Fatalf("mkdir tests: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "a", "a.vox"), []byte("pub fn one() -> i32 { return hidden(); }\nfn hidden() -> i32 { return 1; }\n"), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	// Same-package unit test: can call `hidden()` directly.
	if err := os.WriteFile(filepath.Join(root, "src", "a", "a_test.vox"), []byte("import \"std/testing\" as t\nfn test_unit_private_access() -> () { t.assert_eq(hidden(), 1); }\n"), 0o644); err != nil {
		t.Fatalf("write a_test: %v", err)
	}
	// External test: must use pub API.
	if err := os.WriteFile(filepath.Join(root, "tests", "basic.vox"), []byte("import \"std/testing\" as t\nimport \"a\" as a\nfn test_integration_pub_api() -> () { t.assert_eq(a.one(), 1); }\n"), 0o644); err != nil {
		t.Fatalf("write tests/basic: %v", err)
	}

	// 3) Use stage1 compiler to build+run tests.
	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "test-pkg", outBin)
	cmd.Dir = root
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 test-pkg failed: %v\n%s", err, string(b))
	}
	out := string(b)
	if !strings.Contains(out, "[OK] a::test_unit_private_access") {
		t.Fatalf("missing unit test ok line:\n%s", out)
	}
	if !strings.Contains(out, "[OK] tests::test_integration_pub_api") {
		t.Fatalf("missing integration test ok line:\n%s", out)
	}
	if !strings.Contains(out, "[test] 2 passed, 0 failed") {
		t.Fatalf("unexpected summary:\n%s", out)
	}
}

func TestStage1ToolchainSelfBuildsStage1AndBuildsPackage(t *testing.T) {
	// 1) Build stage1 compiler A (vox_stage1) using stage0.
	stage1Dir := filepath.Clean(filepath.Join("..", "..", "..", "stage1"))
	stage1BinA, err := compile(stage1Dir)
	if err != nil {
		t.Fatalf("build stage1 failed: %v", err)
	}

	stage1DirAbs, err := filepath.Abs(stage1Dir)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}

	// 2) Use stage1 A to self-build stage1 B in stage1Dir/target/debug so __exe_path-based std discovery works.
	outRel := filepath.Join("target", "debug", "vox_stage1_b")
	if err := os.MkdirAll(filepath.Join(stage1DirAbs, "target", "debug"), 0o755); err != nil {
		t.Fatalf("mkdir target/debug: %v", err)
	}
	cmd := exec.Command(stage1BinA, "build-pkg", outRel)
	cmd.Dir = stage1DirAbs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("stage1 A self-build failed: %v", err)
	}
	stage1BinB := filepath.Join(stage1DirAbs, outRel)
	if _, err := os.Stat(stage1BinB); err != nil {
		t.Fatalf("missing stage1 B binary: %v", err)
	}

	// 3) Use stage1 B to build a temp package that uses std + path deps.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src", "a"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Dependency package.
	depRoot := filepath.Join(root, "dep_pkg")
	if err := os.MkdirAll(filepath.Join(depRoot, "src"), 0o755); err != nil {
		t.Fatalf("mkdir dep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "vox.toml"), []byte(`[package]
name = "dep"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write dep vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "src", "dep.vox"), []byte("pub fn two() -> i32 { return 2; }\n"), 0o644); err != nil {
		t.Fatalf("write dep src: %v", err)
	}

	// Root package manifest (uses dep by path).
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "dep_pkg" }
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}

	mainSrc := "import \"std/prelude\" as prelude\nimport \"a\" as a\nimport \"dep\" as dep\nfn main() -> i32 { prelude.assert(true); return a.one() + dep.two(); }\n"
	aSrc := "pub fn one() -> i32 { return 1; }\n"
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte(mainSrc), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "a", "a.vox"), []byte(aSrc), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}

	outBin := filepath.Join(root, "out")
	build := exec.Command(stage1BinB, "build-pkg", outBin)
	build.Dir = root
	b, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 B build-pkg failed: %v\n%s", err, string(b))
	}

	// Run produced binary and check output (driver prints main return).
	run := exec.Command(outBin)
	run.Dir = root
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run built program failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "3" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestStage1ExitCodeNonZeroOnBuildPkgFailure(t *testing.T) {
	// Build stage1 compiler (vox_stage1) using stage0.
	stage1Dir := filepath.Clean(filepath.Join("..", "..", "..", "stage1"))
	stage1Bin, err := compile(stage1Dir)
	if err != nil {
		t.Fatalf("build stage1 failed: %v", err)
	}

	// Create a package with a syntax error so Stage1 reports a compile error
	// via return code (not panic), and ensure the process exit code is non-zero.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}
	// Missing expression after return.
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { return ; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err == nil {
		t.Fatalf("expected stage1 build-pkg to fail with non-zero exit code")
	}
}
