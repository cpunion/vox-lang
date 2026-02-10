package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"voxlang/internal/codegen"
)

const selfhostTestsEnv = "VOX_RUN_SELFHOST_TESTS"

var (
	stage1ToolOnce     sync.Once
	stage1ToolBinPath  string
	stage1ToolBuildErr error

	stage2ToolOnce     sync.Once
	stage2ToolDirAbs   string
	stage2ToolBinPath  string
	stage2ToolBuildErr error
	stage2ToolBuildOut string
)

func stage1ToolDir() string {
	return filepath.Clean(filepath.Join("..", "..", "..", "stage1"))
}

func stage2ToolDir() string {
	return filepath.Clean(filepath.Join("..", "..", "..", "stage2"))
}

func stage1ToolBin(t *testing.T) string {
	t.Helper()
	stage1ToolOnce.Do(func() {
		stage1ToolBinPath, stage1ToolBuildErr = compileWithDriver(stage1ToolDir(), codegen.DriverMainTool)
	})
	if stage1ToolBuildErr != nil {
		t.Fatalf("build stage1 failed: %v", stage1ToolBuildErr)
	}
	return stage1ToolBinPath
}

func requireSelfhostTests(t *testing.T) {
	t.Helper()
	if os.Getenv(selfhostTestsEnv) != "1" {
		t.Skipf("set %s=1 to run self-host tests", selfhostTestsEnv)
	}
}

func stage2ToolBinBuiltByStage1(t *testing.T) (dirAbs string, binPath string) {
	t.Helper()
	stage2ToolOnce.Do(func() {
		// 1) Build stage1 compiler A (tool driver) using stage0.
		stage1BinA := stage1ToolBin(t)

		// 2) Use stage1 A to build stage2 compiler B in stage2/target/debug so
		// __exe_path-based std discovery resolves stage2/src/std.
		stage2Dir := stage2ToolDir()
		stage2DirAbs0, err := filepath.Abs(stage2Dir)
		if err != nil {
			stage2ToolBuildErr = err
			return
		}
		outRel := filepath.Join("target", "debug", "vox_stage2_b_tool")
		if err := os.MkdirAll(filepath.Join(stage2DirAbs0, "target", "debug"), 0o755); err != nil {
			stage2ToolBuildErr = err
			return
		}
		cmd := exec.Command(stage1BinA, "build-pkg", "--driver=tool", outRel)
		cmd.Dir = stage2DirAbs0
		b, err := cmd.CombinedOutput()
		stage2ToolBuildOut = string(b)
		if err != nil {
			stage2ToolBuildErr = err
			return
		}

		stage2BinB := filepath.Join(stage2DirAbs0, outRel)
		if _, err := os.Stat(stage2BinB); err != nil {
			stage2ToolBuildErr = err
			return
		}
		stage2ToolDirAbs = stage2DirAbs0
		stage2ToolBinPath = stage2BinB
	})
	if stage2ToolBuildErr != nil {
		t.Fatalf("build stage2 tool failed: %v\n%s", stage2ToolBuildErr, stage2ToolBuildOut)
	}
	return stage2ToolDirAbs, stage2ToolBinPath
}

func TestStage1ToolchainBuildsMultiModuleProgram(t *testing.T) {
	t.Parallel()

	// 1) Build the stage1 compiler (vox_stage1) using stage0.
	stage1Bin := stage1ToolBin(t)

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

func TestStage1ToolchainEmitCAndBuildCommands(t *testing.T) {
	t.Parallel()

	// Build stage1 compiler (vox_stage1) using stage0 (tool driver).
	stage1Bin := stage1ToolBin(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { return 7; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// emit-c should write C source successfully.
	outC := filepath.Join(root, "out.c")
	emit := exec.Command(stage1Bin, "emit-c", outC, "src/main.vox")
	emit.Dir = root
	if b, err := emit.CombinedOutput(); err != nil {
		t.Fatalf("stage1 emit-c failed: %v\n%s", err, string(b))
	}
	csrc, err := os.ReadFile(outC)
	if err != nil {
		t.Fatalf("read emitted C: %v", err)
	}
	if !strings.Contains(string(csrc), "vox_fn_mmain") {
		t.Fatalf("unexpected emitted C content:\n%s", string(csrc))
	}

	// build (default user driver): running binary prints main return value.
	outUser := filepath.Join(root, "out_user")
	buildUser := exec.Command(stage1Bin, "build", outUser, "src/main.vox")
	buildUser.Dir = root
	if b, err := buildUser.CombinedOutput(); err != nil {
		t.Fatalf("stage1 build failed: %v\n%s", err, string(b))
	}
	runUser := exec.Command(outUser)
	runUser.Dir = root
	out, err := runUser.CombinedOutput()
	if err != nil {
		t.Fatalf("run user binary failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "7" {
		t.Fatalf("unexpected user driver output: %q", got)
	}

	// build --driver=tool: running binary should be quiet and return exit code.
	outTool := filepath.Join(root, "out_tool")
	buildTool := exec.Command(stage1Bin, "build", "--driver=tool", outTool, "src/main.vox")
	buildTool.Dir = root
	if b, err := buildTool.CombinedOutput(); err != nil {
		t.Fatalf("stage1 build --driver=tool failed: %v\n%s", err, string(b))
	}
	runTool := exec.Command(outTool)
	runTool.Dir = root
	out2, err := runTool.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit status for tool binary")
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if ee.ExitCode() != 7 {
		t.Fatalf("unexpected tool driver exit code: %d", ee.ExitCode())
	}
	if got := strings.TrimSpace(string(out2)); got != "" {
		t.Fatalf("expected no stdout for tool driver, got: %q", got)
	}
}

func TestStage1ToolchainBuildPkgAndTestPkgUseLocalStd(t *testing.T) {
	t.Parallel()

	// Build stage1 compiler (vox_stage1) using stage0 (tool driver).
	stage1Bin := stage1ToolBin(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src", "std", "prelude"), 0o755); err != nil {
		t.Fatalf("mkdir prelude: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src", "std", "testing"), 0o755); err != nil {
		t.Fatalf("mkdir testing: %v", err)
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

	// Local std/prelude defines a marker not present in embedded prelude.
	// If build-pkg incorrectly injects embedded std, this program will fail to typecheck.
	preludeAssert := `pub fn assert(cond: bool) -> () { if !cond { panic("assertion failed"); } }
pub fn marker() -> i32 { return 11; }
`
	if err := os.WriteFile(filepath.Join(root, "src", "std", "prelude", "assert.vox"), []byte(preludeAssert), 0o644); err != nil {
		t.Fatalf("write local prelude: %v", err)
	}
	localTesting := `import "std/prelude" as prelude
pub fn assert(cond: bool) -> () { prelude.assert(cond); }
`
	if err := os.WriteFile(filepath.Join(root, "src", "std", "testing", "testing.vox"), []byte(localTesting), 0o644); err != nil {
		t.Fatalf("write local testing: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("import \"std/prelude\" as prelude\nfn main() -> i32 { return prelude.marker(); }\n"), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "tests", "basic.vox"), []byte("import \"std/testing\" as t\nfn test_local_std() -> () { t.assert(true); }\n"), 0o644); err != nil {
		t.Fatalf("write test: %v", err)
	}

	// build-pkg should use local std and produce a runnable binary.
	outBin := filepath.Join(root, "out")
	build := exec.Command(stage1Bin, "build-pkg", outBin)
	build.Dir = root
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("stage1 build-pkg failed: %v\n%s", err, string(b))
	}
	run := exec.Command(outBin)
	run.Dir = root
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run built binary failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "11" {
		t.Fatalf("unexpected output: %q", got)
	}

	// test-pkg should also use local std/testing and run tests successfully.
	testCmd := exec.Command(stage1Bin, "test-pkg", outBin)
	testCmd.Dir = root
	tb, err := testCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 test-pkg failed: %v\n%s", err, string(tb))
	}
	ts := string(tb)
	if !strings.Contains(ts, "[OK] tests::test_local_std") {
		t.Fatalf("missing test ok line:\n%s", ts)
	}
	if !strings.Contains(ts, "[test] 1 passed, 0 failed") {
		t.Fatalf("unexpected test summary:\n%s", ts)
	}
}

func TestStage1CliArgumentValidation(t *testing.T) {
	t.Parallel()

	// Build stage1 compiler (vox_stage1) using stage0 (tool driver).
	stage1Bin := stage1ToolBin(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Unknown command should fail with non-zero code.
	cmdUnknown := exec.Command(stage1Bin, "nope", filepath.Join(root, "out"))
	cmdUnknown.Dir = root
	bUnknown, err := cmdUnknown.CombinedOutput()
	if err == nil {
		t.Fatalf("expected unknown command to fail")
	}
	if !strings.Contains(string(bUnknown), "unknown command") {
		t.Fatalf("expected unknown command output, got:\n%s", string(bUnknown))
	}

	// Unknown driver value should fail.
	cmdDriver := exec.Command(stage1Bin, "build", "--driver=bad", filepath.Join(root, "out"), "src/main.vox")
	cmdDriver.Dir = root
	bDriver, err := cmdDriver.CombinedOutput()
	if err == nil {
		t.Fatalf("expected unknown driver to fail")
	}
	if !strings.Contains(string(bDriver), "unknown driver") {
		t.Fatalf("expected unknown driver output, got:\n%s", string(bDriver))
	}

	// Missing source list should fail for emit-c/build.
	cmdEmitMissing := exec.Command(stage1Bin, "emit-c", filepath.Join(root, "out.c"))
	cmdEmitMissing.Dir = root
	bEmit, err := cmdEmitMissing.CombinedOutput()
	if err == nil {
		t.Fatalf("expected emit-c missing sources to fail")
	}
	if !strings.Contains(string(bEmit), "missing sources") {
		t.Fatalf("expected missing sources output (emit-c), got:\n%s", string(bEmit))
	}

	cmdBuildMissing := exec.Command(stage1Bin, "build", filepath.Join(root, "out"))
	cmdBuildMissing.Dir = root
	bBuild, err := cmdBuildMissing.CombinedOutput()
	if err == nil {
		t.Fatalf("expected build missing sources to fail")
	}
	if !strings.Contains(string(bBuild), "missing sources") {
		t.Fatalf("expected missing sources output (build), got:\n%s", string(bBuild))
	}
}

func TestStage1ToolchainBuildsWithTransitivePathDeps(t *testing.T) {
	t.Parallel()

	// 1) Build the stage1 compiler (vox_stage1) using stage0 (tool driver).
	stage1Bin := stage1ToolBin(t)

	// 2) Create a package with transitive path dependencies:
	// app -> dep -> b
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// b package.
	bRoot := filepath.Join(root, "b_pkg")
	if err := os.MkdirAll(filepath.Join(bRoot, "src"), 0o755); err != nil {
		t.Fatalf("mkdir b: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bRoot, "vox.toml"), []byte(`[package]
name = "b"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write b vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bRoot, "src", "b.vox"), []byte("pub fn one() -> i32 { return 1; }\n"), 0o644); err != nil {
		t.Fatalf("write b src: %v", err)
	}

	// dep package depends on b.
	depRoot := filepath.Join(root, "dep_pkg")
	if err := os.MkdirAll(filepath.Join(depRoot, "src"), 0o755); err != nil {
		t.Fatalf("mkdir dep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "vox.toml"), []byte(`[package]
name = "dep"
version = "0.1.0"
edition = "2026"

[dependencies]
b = { path = "../b_pkg" }
`), 0o644); err != nil {
		t.Fatalf("write dep vox.toml: %v", err)
	}
	depSrc := "import \"b\" as b\npub fn two() -> i32 { return 1 + b.one(); }\n"
	if err := os.WriteFile(filepath.Join(depRoot, "src", "dep.vox"), []byte(depSrc), 0o644); err != nil {
		t.Fatalf("write dep src: %v", err)
	}

	// Root package depends on dep.
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "dep_pkg" }
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}
	mainSrc := "import \"dep\" as dep\nfn main() -> i32 { return dep.two(); }\n"
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte(mainSrc), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}

	// 3) Use stage1 compiler to build it; transitive deps must be loadable.
	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 build-pkg failed: %v\n%s", err, string(b))
	}

	// 4) Run the produced binary and check output (driver prints main return).
	run := exec.Command(outBin)
	run.Dir = root
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run built program failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "2" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestStage1BuildPkgNoSymbolCollisionBetweenQualifiedAndPlainNames(t *testing.T) {
	t.Parallel()

	// Build stage1 compiler (vox_stage1) using stage0 (tool driver).
	stage1Bin := stage1ToolBin(t)

	// root package with a path dep.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// dep package.
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
	if err := os.WriteFile(filepath.Join(depRoot, "src", "dep.vox"), []byte("pub fn one() -> i32 { return 2; }\n"), 0o644); err != nil {
		t.Fatalf("write dep src: %v", err)
	}

	// Root package manifest.
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "dep_pkg" }
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}

	// This used to be a potential collision in C backends if mangling wasn't collision-free:
	// - local function: dep__one
	// - dep package function: pkg.dep::one
	mainSrc := "import \"dep\" as dep\nfn dep__one() -> i32 { return 1; }\nfn main() -> i32 { return dep__one() + dep.one(); }\n"
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte(mainSrc), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stage1 build-pkg failed: %v\n%s", err, string(b))
	}

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

func TestStage1BuildPkgImportSchemesDisambiguateDepAndLocalModule(t *testing.T) {
	t.Parallel()

	// Build stage1 compiler (vox_stage1) using stage0 (tool driver).
	stage1Bin := stage1ToolBin(t)

	// Root package with a local module "dep" and a dependency package also named "dep".
	// Plain `import "dep"` must be ambiguous and require `pkg:`/`mod:`.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src", "dep"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Dependency package: dep.
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
	if err := os.WriteFile(filepath.Join(depRoot, "src", "dep.vox"), []byte("pub fn one() -> i32 { return 1; }\n"), 0o644); err != nil {
		t.Fatalf("write dep src: %v", err)
	}

	// Local module: dep.
	if err := os.WriteFile(filepath.Join(root, "src", "dep", "dep.vox"), []byte("pub fn one() -> i32 { return 100; }\n"), 0o644); err != nil {
		t.Fatalf("write local dep module: %v", err)
	}

	// Root package manifest depends on dep by path.
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "dep_pkg" }
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}

	outBin := filepath.Join(root, "out")

	// 1) Plain import should fail due to ambiguity.
	ambiguousMain := "import \"dep\" as dep\nfn main() -> i32 { return dep.one(); }\n"
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte(ambiguousMain), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	b, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected stage1 build-pkg to fail on ambiguous import")
	}
	if !strings.Contains(string(b), "ambiguous import") {
		t.Fatalf("expected ambiguous import output, got:\n%s", string(b))
	}

	// 2) pkg:dep should resolve to the dependency package.
	pkgMain := "import \"pkg:dep\" as dep\nfn main() -> i32 { return dep.one(); }\n"
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte(pkgMain), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	cmd2 := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd2.Dir = root
	if b2, err := cmd2.CombinedOutput(); err != nil {
		t.Fatalf("stage1 build-pkg failed: %v\n%s", err, string(b2))
	}
	run := exec.Command(outBin)
	run.Dir = root
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run built program failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "1" {
		t.Fatalf("unexpected output: %q", got)
	}

	// 3) mod:dep should resolve to the local module.
	modMain := "import \"mod:dep\" as dep\nfn main() -> i32 { return dep.one(); }\n"
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte(modMain), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	cmd3 := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd3.Dir = root
	if b3, err := cmd3.CombinedOutput(); err != nil {
		t.Fatalf("stage1 build-pkg failed: %v\n%s", err, string(b3))
	}
	run2 := exec.Command(outBin)
	run2.Dir = root
	out2, err := run2.CombinedOutput()
	if err != nil {
		t.Fatalf("run built program failed: %v\n%s", err, string(out2))
	}
	if got := strings.TrimSpace(string(out2)); got != "100" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestStage1BuildPkgFailsOnInvalidManifest(t *testing.T) {
	t.Parallel()

	// Build stage1 compiler (vox_stage1) using stage0 (tool driver).
	stage1Bin := stage1ToolBin(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Invalid manifest line.
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "../dep" }
this is not valid
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	b, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected stage1 build-pkg to fail on invalid vox.toml")
	}
	if !strings.Contains(string(b), "invalid vox.toml") {
		t.Fatalf("expected invalid vox.toml output, got:\n%s", string(b))
	}
}

func TestStage1BuildPkgFailsOnDuplicateDependencyName(t *testing.T) {
	t.Parallel()

	// Build stage1 compiler (vox_stage1) using stage0 (tool driver).
	stage1Bin := stage1ToolBin(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Duplicate dependency name (stage1 manifest parser is line-based).
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "dep1" }
dep = { path = "dep2" }
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	b, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected stage1 build-pkg to fail on duplicate deps")
	}
	if !strings.Contains(string(b), "duplicate dependency") {
		t.Fatalf("expected duplicate dependency output, got:\n%s", string(b))
	}
}

func TestStage1BuildPkgWritesLockfile(t *testing.T) {
	t.Parallel()

	stage1Bin := stage1ToolBin(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir root src: %v", err)
	}
	depRoot := filepath.Join(root, "dep_pkg")
	if err := os.MkdirAll(filepath.Join(depRoot, "src"), 0o755); err != nil {
		t.Fatalf("mkdir dep src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "vox.toml"), []byte(`[package]
name = "dep"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write dep vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "src", "dep.vox"), []byte("pub fn two() -> i32 { return 2; }\n"), 0o644); err != nil {
		t.Fatalf("write dep source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "dep_pkg", version = "0.1.0" }
`), 0o644); err != nil {
		t.Fatalf("write root vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("import \"dep\" as dep\nfn main() -> i32 { return dep.two(); }\n"), 0o644); err != nil {
		t.Fatalf("write root source: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stage1 build-pkg failed: %v\n%s", err, string(b))
	}

	lockPath := filepath.Join(root, "vox.lock")
	lockBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read vox.lock: %v", err)
	}
	lock := string(lockBytes)
	if !strings.Contains(lock, "[[dependency]]") {
		t.Fatalf("expected lockfile dependency section, got:\n%s", lock)
	}
	if !strings.Contains(lock, `name = "dep"`) {
		t.Fatalf("expected dep name in lockfile, got:\n%s", lock)
	}
	if !strings.Contains(lock, `source = "path"`) {
		t.Fatalf("expected path source in lockfile, got:\n%s", lock)
	}
	if !strings.Contains(lock, "dep_pkg") {
		t.Fatalf("expected dep path in lockfile, got:\n%s", lock)
	}
	if !strings.Contains(lock, "resolved_path") {
		t.Fatalf("expected resolved_path in lockfile, got:\n%s", lock)
	}
	if !strings.Contains(lock, `digest = "`) {
		t.Fatalf("expected digest in lockfile, got:\n%s", lock)
	}
}

func TestStage1BuildPkgFailsWhenLockDigestMismatch(t *testing.T) {
	t.Parallel()

	stage1Bin := stage1ToolBin(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir root src: %v", err)
	}
	depRoot := filepath.Join(root, "dep_pkg")
	if err := os.MkdirAll(filepath.Join(depRoot, "src"), 0o755); err != nil {
		t.Fatalf("mkdir dep src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "vox.toml"), []byte(`[package]
name = "dep"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write dep vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "src", "dep.vox"), []byte("pub fn two() -> i32 { return 2; }\n"), 0o644); err != nil {
		t.Fatalf("write dep source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "dep_pkg", version = "0.1.0" }
`), 0o644); err != nil {
		t.Fatalf("write root vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("import \"dep\" as dep\nfn main() -> i32 { return dep.two(); }\n"), 0o644); err != nil {
		t.Fatalf("write root source: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stage1 build-pkg first run failed: %v\n%s", err, string(b))
	}

	// Tamper dependency content without updating lockfile.
	if err := os.WriteFile(filepath.Join(depRoot, "src", "dep.vox"), []byte("pub fn two() -> i32 { return 3; }\n"), 0o644); err != nil {
		t.Fatalf("rewrite dep source: %v", err)
	}

	cmd2 := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd2.Dir = root
	out, err := cmd2.CombinedOutput()
	if err == nil {
		t.Fatalf("expected lock mismatch failure after dependency tamper")
	}
	s := string(out)
	if !strings.Contains(s, "invalid vox.lock") {
		t.Fatalf("expected invalid vox.lock error, got:\n%s", s)
	}
	if !strings.Contains(s, "dependency mismatch") {
		t.Fatalf("expected dependency mismatch detail, got:\n%s", s)
	}
}

func TestStage1BuildPkgFailsOnMissingRegistryDependency(t *testing.T) {
	t.Parallel()

	stage1Bin := stage1ToolBin(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = "1.2.3"
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	b, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected stage1 build-pkg to fail when registry dependency is missing")
	}
	out := string(b)
	if !strings.Contains(out, "invalid vox.toml") {
		t.Fatalf("expected invalid vox.toml message, got:\n%s", out)
	}
	if !strings.Contains(out, "registry dependency not found") {
		t.Fatalf("expected registry dependency error detail, got:\n%s", out)
	}
}

func TestStage1BuildPkgSupportsVersionDependencyFromRegistryCache(t *testing.T) {
	t.Parallel()

	stage1Bin := stage1ToolBin(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	regDep := filepath.Join(root, ".vox", "deps", "registry", "dep", "1.2.3")
	if err := os.MkdirAll(filepath.Join(regDep, "src"), 0o755); err != nil {
		t.Fatalf("mkdir registry dep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(regDep, "vox.toml"), []byte(`[package]
name = "dep"
version = "1.2.3"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write dep vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(regDep, "src", "dep.vox"), []byte("pub fn two() -> i32 { return 2; }\n"), 0o644); err != nil {
		t.Fatalf("write dep source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = "1.2.3"
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("import \"dep\" as dep\nfn main() -> i32 { return dep.two(); }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stage1 build-pkg failed: %v\n%s", err, string(b))
	}
	run := exec.Command(outBin)
	got, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run output failed: %v\n%s", err, string(got))
	}
	if strings.TrimSpace(string(got)) != "2" {
		t.Fatalf("unexpected output: %q", string(got))
	}
}

func TestStage1BuildPkgSupportsGitDependency(t *testing.T) {
	t.Parallel()

	stage1Bin := stage1ToolBin(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	depRepo := filepath.Join(root, "dep_git_repo")
	if err := os.MkdirAll(filepath.Join(depRepo, "src"), 0o755); err != nil {
		t.Fatalf("mkdir dep repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRepo, "vox.toml"), []byte(`[package]
name = "dep"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write dep vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRepo, "src", "dep.vox"), []byte("pub fn one() -> i32 { return 1; }\n"), 0o644); err != nil {
		t.Fatalf("write dep source: %v", err)
	}
	init := exec.Command("git", "init")
	init.Dir = depRepo
	if b, err := init.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, string(b))
	}
	add := exec.Command("git", "add", ".")
	add.Dir = depRepo
	if b, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, string(b))
	}
	commit := exec.Command("git", "-c", "user.email=vox@example.com", "-c", "user.name=vox", "commit", "-m", "init")
	commit.Dir = depRepo
	if b, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, string(b))
	}

	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte(`[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { git = "dep_git_repo" }
`), 0o644); err != nil {
		t.Fatalf("write vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("import \"dep\" as dep\nfn main() -> i32 { return dep.one() + 2; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stage1 build-pkg failed: %v\n%s", err, string(b))
	}
	run := exec.Command(outBin)
	got, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run output failed: %v\n%s", err, string(got))
	}
	if strings.TrimSpace(string(got)) != "3" {
		t.Fatalf("unexpected output: %q", string(got))
	}
	lockPath := filepath.Join(root, "vox.lock")
	lockBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read vox.lock: %v", err)
	}
	lock := string(lockBytes)
	if !strings.Contains(lock, `source = "git"`) {
		t.Fatalf("expected git source in lockfile, got:\n%s", lock)
	}
	if !strings.Contains(lock, `rev = "`) {
		t.Fatalf("expected resolved git rev in lockfile, got:\n%s", lock)
	}
}

func TestStage1ToolchainTestPkgRunsTests(t *testing.T) {
	t.Parallel()

	// 1) Build the stage1 compiler (vox_stage1) using stage0.
	stage1Bin := stage1ToolBin(t)

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
	requireSelfhostTests(t)
	// 1) Build stage1 compiler A (vox_stage1) using stage0.
	stage1Dir := stage1ToolDir()
	stage1BinA := stage1ToolBin(t)

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

func TestStage1SelfBuiltCompilerIsQuietOnSuccess(t *testing.T) {
	requireSelfhostTests(t)
	// 1) Build stage1 compiler A (vox_stage1) using stage0 (tool driver).
	stage1Dir := stage1ToolDir()
	stage1BinA := stage1ToolBin(t)

	stage1DirAbs, err := filepath.Abs(stage1Dir)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}

	// 2) Use stage1 A to self-build stage1 B as a *tool* binary (quiet, exit-code based).
	outRel := filepath.Join("target", "debug", "vox_stage1_b_tool")
	if err := os.MkdirAll(filepath.Join(stage1DirAbs, "target", "debug"), 0o755); err != nil {
		t.Fatalf("mkdir target/debug: %v", err)
	}
	cmd := exec.Command(stage1BinA, "build-pkg", "--driver=tool", outRel)
	cmd.Dir = stage1DirAbs
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 A self-build (tool) failed: %v\n%s", err, string(b))
	}

	stage1BinB := filepath.Join(stage1DirAbs, outRel)
	if _, err := os.Stat(stage1BinB); err != nil {
		t.Fatalf("missing stage1 B tool binary: %v", err)
	}

	// 3) Running stage1 B successfully should not print a trailing "0".
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
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	outBin := filepath.Join(root, "out")
	cmd2 := exec.Command(stage1BinB, "build-pkg", outBin)
	cmd2.Dir = root
	b2, err := cmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 B build-pkg failed: %v\n%s", err, string(b2))
	}
	if got := strings.TrimSpace(string(b2)); got != "" {
		t.Fatalf("expected no output on success, got:\n%s", got)
	}
}

func TestStage1BuildsStage2AndBuildsPackage(t *testing.T) {
	requireSelfhostTests(t)

	_, stage2BinB := stage2ToolBinBuiltByStage1(t)

	// Use stage2 B to build and run a tiny package.
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
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("import \"std/prelude\" as prelude\nfn main() -> i32 { prelude.assert(true); return 7; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	outBin := filepath.Join(root, "out")
	cmd2 := exec.Command(stage2BinB, "build-pkg", outBin)
	cmd2.Dir = root
	b2, err := cmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 build-pkg failed: %v\n%s", err, string(b2))
	}
	run := exec.Command(outBin)
	run.Dir = root
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run built program failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "7" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestStage1BuildsStage2AndRunsStage2Tests(t *testing.T) {
	requireSelfhostTests(t)

	stage2DirAbs, stage2BinB := stage2ToolBinBuiltByStage1(t)
	outRel := filepath.Join("target", "debug", "vox_stage2.test")
	stage2BinName := filepath.Base(stage2BinB)

	// Usage should reflect current binary name instead of hardcoded stage1 name.
	cmdUsage := exec.Command(stage2BinB)
	cmdUsage.Dir = stage2DirAbs
	bu, err := cmdUsage.CombinedOutput()
	if err == nil {
		t.Fatalf("expected stage2 usage invocation without command to fail")
	}
	if !strings.Contains(string(bu), stage2BinName+" usage:") {
		t.Fatalf("expected dynamic usage header with stage2 binary name, got:\n%s", string(bu))
	}

	// Ensure test selection flags are available in stage2 test-pkg.
	cmdList := exec.Command(stage2BinB, "test-pkg", "--filter=std_sync_runtime_generic_api_smoke", "--list", outRel)
	cmdList.Dir = stage2DirAbs
	bl, err := cmdList.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --list failed: %v\n%s", err, string(bl))
	}
	if !strings.Contains(string(bl), "[select] discovered:") {
		t.Fatalf("expected stage2 selection summary, got:\n%s", string(bl))
	}
	if !strings.Contains(string(bl), "[test] test_std_sync_runtime_generic_api_smoke") {
		t.Fatalf("expected filtered test in list output, got:\n%s", string(bl))
	}
	if strings.Contains(string(bl), "[test] test_std_testing_smoke") {
		t.Fatalf("expected filter to exclude unrelated tests, got:\n%s", string(bl))
	}

	cmdRun := exec.Command(stage2BinB, "test-pkg", "--run=*std_testing*", "--list", outRel)
	cmdRun.Dir = stage2DirAbs
	brun, err := cmdRun.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --run failed: %v\n%s", err, string(brun))
	}
	if !strings.Contains(string(brun), "[test] test_std_testing_smoke") {
		t.Fatalf("expected run pattern to select std testing smoke, got:\n%s", string(brun))
	}
	cmdRunSplit := exec.Command(stage2BinB, "test-pkg", "--run", "*std_testing*", "--list", outRel)
	cmdRunSplit.Dir = stage2DirAbs
	brunSplit, err := cmdRunSplit.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --run <value> failed: %v\n%s", err, string(brunSplit))
	}
	if !strings.Contains(string(brunSplit), "[test] test_std_testing_smoke") {
		t.Fatalf("expected split run pattern to select std testing smoke, got:\n%s", string(brunSplit))
	}
	cmdModule := exec.Command(stage2BinB, "test-pkg", "--module=typecheck", "--run=*typecheck_allows_generic_fn_sig", "--list", outRel)
	cmdModule.Dir = stage2DirAbs
	bmod, err := cmdModule.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --module failed: %v\n%s", err, string(bmod))
	}
	if !strings.Contains(string(bmod), "[select] --module: \"typecheck\"") {
		t.Fatalf("expected module filter in text selection output, got:\n%s", string(bmod))
	}
	if !strings.Contains(string(bmod), "[test] typecheck::test_typecheck_allows_generic_fn_sig") {
		t.Fatalf("expected module-filtered test in list output, got:\n%s", string(bmod))
	}
	if strings.Contains(string(bmod), "[test] parse::test_parse_single_fn_return_int") {
		t.Fatalf("expected module filter to exclude parse tests, got:\n%s", string(bmod))
	}
	cmdModuleSplit := exec.Command(stage2BinB, "test-pkg", "--module", "typecheck", "--run", "*typecheck_allows_generic_fn_sig", "--list", outRel)
	cmdModuleSplit.Dir = stage2DirAbs
	bmodSplit, err := cmdModuleSplit.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --module <value> failed: %v\n%s", err, string(bmodSplit))
	}
	if !strings.Contains(string(bmodSplit), "[select] --module: \"typecheck\"") {
		t.Fatalf("expected split module filter in selection output, got:\n%s", string(bmodSplit))
	}
	cmdFilterSplit := exec.Command(stage2BinB, "test-pkg", "--filter", "std_sync_runtime_generic_api_smoke", "--list", outRel)
	cmdFilterSplit.Dir = stage2DirAbs
	bfilterSplit, err := cmdFilterSplit.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --filter <value> failed: %v\n%s", err, string(bfilterSplit))
	}
	if !strings.Contains(string(bfilterSplit), "[test] test_std_sync_runtime_generic_api_smoke") {
		t.Fatalf("expected split filter to select target test, got:\n%s", string(bfilterSplit))
	}

	cmdJobs := exec.Command(stage2BinB, "test-pkg", "--jobs=2", "--run=*std_sync_runtime_generic_api_smoke", "--list", outRel)
	cmdJobs.Dir = stage2DirAbs
	bjobs, err := cmdJobs.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --jobs failed: %v\n%s", err, string(bjobs))
	}
	if !strings.Contains(string(bjobs), "[select] --jobs: 2") {
		t.Fatalf("expected jobs selection in text output, got:\n%s", string(bjobs))
	}
	cmdJobsShort := exec.Command(stage2BinB, "test-pkg", "-j", "2", "--run=*std_sync_runtime_generic_api_smoke", "--list", outRel)
	cmdJobsShort.Dir = stage2DirAbs
	bjobsShort, err := cmdJobsShort.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg -j failed: %v\n%s", err, string(bjobsShort))
	}
	if !strings.Contains(string(bjobsShort), "[select] --jobs: 2") {
		t.Fatalf("expected short -j selection in text output, got:\n%s", string(bjobsShort))
	}
	cmdJobsBad := exec.Command(stage2BinB, "test-pkg", "--jobs=0", "--list", outRel)
	cmdJobsBad.Dir = stage2DirAbs
	bjobsBad, err := cmdJobsBad.CombinedOutput()
	if err == nil {
		t.Fatalf("expected stage2 test-pkg --jobs=0 to fail, got:\n%s", string(bjobsBad))
	}
	if !strings.Contains(string(bjobsBad), "invalid --jobs value") {
		t.Fatalf("expected invalid --jobs diagnostic, got:\n%s", string(bjobsBad))
	}

	cmdJSON := exec.Command(stage2BinB, "test-pkg", "--jobs=2", "--run=*std_sync_runtime_generic_api_smoke", "--list", "--json", outRel)
	cmdJSON.Dir = stage2DirAbs
	bj, err := cmdJSON.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --json failed: %v\n%s", err, string(bj))
	}
	if !strings.Contains(string(bj), "\"list_only\":true") {
		t.Fatalf("expected json list report, got:\n%s", string(bj))
	}
	if !strings.Contains(string(bj), "\"selected_tests\":[\"test_std_sync_runtime_generic_api_smoke\"]") {
		t.Fatalf("expected selected test in json output, got:\n%s", string(bj))
	}
	if !strings.Contains(string(bj), "\"jobs\":2") {
		t.Fatalf("expected jobs in selection json output, got:\n%s", string(bj))
	}
	cmdFailFastList := exec.Command(stage2BinB, "test-pkg", "--fail-fast", "--run=*std_sync_runtime_generic_api_smoke", "--list", outRel)
	cmdFailFastList.Dir = stage2DirAbs
	bffList, err := cmdFailFastList.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --fail-fast --list failed: %v\n%s", err, string(bffList))
	}
	if !strings.Contains(string(bffList), "[select] --fail-fast: true") {
		t.Fatalf("expected fail-fast selection in text output, got:\n%s", string(bffList))
	}
	cmdFailFastJSON := exec.Command(stage2BinB, "test-pkg", "--fail-fast", "--run=*std_sync_runtime_generic_api_smoke", "--list", "--json", outRel)
	cmdFailFastJSON.Dir = stage2DirAbs
	bffJSON, err := cmdFailFastJSON.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --fail-fast --json failed: %v\n%s", err, string(bffJSON))
	}
	if !strings.Contains(string(bffJSON), "\"fail_fast\":true") {
		t.Fatalf("expected fail_fast in selection json output, got:\n%s", string(bffJSON))
	}
	cmdJSONModule := exec.Command(stage2BinB, "test-pkg", "--module=typecheck", "--run=*typecheck_allows_generic_fn_sig", "--list", "--json", outRel)
	cmdJSONModule.Dir = stage2DirAbs
	bjm, err := cmdJSONModule.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --module --json failed: %v\n%s", err, string(bjm))
	}
	if !strings.Contains(string(bjm), "\"module\":\"typecheck\"") {
		t.Fatalf("expected module in selection json output, got:\n%s", string(bjm))
	}
	if !strings.Contains(string(bjm), "\"selected_tests\":[\"typecheck::test_typecheck_allows_generic_fn_sig\"]") {
		t.Fatalf("expected module-filtered selected test in json output, got:\n%s", string(bjm))
	}

	cmdJSONRun := exec.Command(stage2BinB, "test-pkg", "--run=*std_sync_runtime_generic_api_smoke", "--json", outRel)
	cmdJSONRun.Dir = stage2DirAbs
	bjr, err := cmdJSONRun.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --json run failed: %v\n%s", err, string(bjr))
	}
	if !strings.Contains(string(bjr), "\"results\"") || !strings.Contains(string(bjr), "\"module_details\"") || !strings.Contains(string(bjr), "\"summary\"") {
		t.Fatalf("expected json run report fields, got:\n%s", string(bjr))
	}
	if !strings.Contains(string(bjr), "\"slowest\"") || !strings.Contains(string(bjr), "\"duration_us\"") {
		t.Fatalf("expected json timing fields, got:\n%s", string(bjr))
	}

	cmd := exec.Command(stage2BinB, "test-pkg", outRel)
	cmd.Dir = stage2DirAbs
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg failed: %v\n%s", err, string(b))
	}
	if !strings.Contains(string(b), "[test]") {
		t.Fatalf("expected stage2 test summary, got:\n%s", string(b))
	}
	if !strings.Contains(string(b), "[module]") {
		t.Fatalf("expected stage2 module summary, got:\n%s", string(b))
	}
	if !strings.Contains(string(b), "[slowest]") || !strings.Contains(string(b), "[time] total:") {
		t.Fatalf("expected stage2 timing summary, got:\n%s", string(b))
	}
	cachePath := filepath.Join(stage2DirAbs, "target", "debug", ".vox_last_failed_tests")
	cache, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read stage2 failed-tests cache: %v", err)
	}
	if !strings.Contains(string(cache), "\"failed_tests\":") {
		t.Fatalf("expected json failed-tests cache, got:\n%s", string(cache))
	}

	cmdRerun := exec.Command(stage2BinB, "test-pkg", "--rerun-failed", outRel)
	cmdRerun.Dir = stage2DirAbs
	br, err := cmdRerun.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 test-pkg --rerun-failed failed: %v\n%s", err, string(br))
	}
	if !strings.Contains(string(br), "no previous failed tests") {
		t.Fatalf("expected no previous failed tests message, got:\n%s", string(br))
	}

	// Ensure failed test logs are still visible when running with module-level jobs.
	failRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(failRoot, "src"), 0o755); err != nil {
		t.Fatalf("mkdir fail src: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(failRoot, "tests"), 0o755); err != nil {
		t.Fatalf("mkdir fail tests: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(failRoot, "target", "debug"), 0o755); err != nil {
		t.Fatalf("mkdir fail target/debug: %v", err)
	}
	if err := os.WriteFile(filepath.Join(failRoot, "vox.toml"), []byte(`[package]
name = "fail_app"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write fail vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(failRoot, "src", "main.vox"), []byte("fn main() -> i32 { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write fail main: %v", err)
	}
	if err := os.WriteFile(filepath.Join(failRoot, "tests", "fail.vox"), []byte("import \"std/testing\" as t\nfn test_fail_marker() -> () { t.fail(\"STAGE2_FAIL_MARKER\"); }\n"), 0o644); err != nil {
		t.Fatalf("write fail test: %v", err)
	}
	failOutRel := filepath.Join("target", "debug", "vox_stage2_fail.test")
	cmdFail := exec.Command(stage2BinB, "test-pkg", "--jobs=2", failOutRel)
	cmdFail.Dir = failRoot
	bf, err := cmdFail.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failing stage2 test-pkg to return non-zero, got success:\n%s", string(bf))
	}
	if !strings.Contains(string(bf), "[FAIL] tests::test_fail_marker") {
		t.Fatalf("expected fail line in stage2 output, got:\n%s", string(bf))
	}
	if !strings.Contains(string(bf), "STAGE2_FAIL_MARKER") {
		t.Fatalf("expected captured failed test log in stage2 output, got:\n%s", string(bf))
	}
	if !strings.Contains(string(bf), "[hint] rerun failed:") || !strings.Contains(string(bf), "--rerun-failed") {
		t.Fatalf("expected rerun hint in stage2 output, got:\n%s", string(bf))
	}
	if !strings.Contains(string(bf), "--jobs=2") {
		t.Fatalf("expected rerun hint to preserve jobs in stage2 output, got:\n%s", string(bf))
	}

	// Ensure fail-fast stops scheduling remaining tests and reports skipped count.
	failFastRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(failFastRoot, "src"), 0o755); err != nil {
		t.Fatalf("mkdir fail-fast src: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(failFastRoot, "tests"), 0o755); err != nil {
		t.Fatalf("mkdir fail-fast tests: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(failFastRoot, "target", "debug"), 0o755); err != nil {
		t.Fatalf("mkdir fail-fast target/debug: %v", err)
	}
	if err := os.WriteFile(filepath.Join(failFastRoot, "vox.toml"), []byte(`[package]
name = "fail_fast_app"
version = "0.1.0"
edition = "2026"
`), 0o644); err != nil {
		t.Fatalf("write fail-fast vox.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(failFastRoot, "src", "main.vox"), []byte("fn main() -> i32 { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write fail-fast main: %v", err)
	}
	if err := os.WriteFile(filepath.Join(failFastRoot, "tests", "fail_fast.vox"), []byte(`import "std/testing" as t
fn test_a_fail() -> () { t.fail("STAGE2_FAIL_FAST_MARKER"); }
fn test_b_should_skip() -> () { t.assert(true); }
`), 0o644); err != nil {
		t.Fatalf("write fail-fast test: %v", err)
	}
	failFastOutRel := filepath.Join("target", "debug", "vox_stage2_fail_fast.test")
	cmdFailFast := exec.Command(stage2BinB, "test-pkg", "--fail-fast", "--jobs=1", "--json", failFastOutRel)
	cmdFailFast.Dir = failFastRoot
	bff, err := cmdFailFast.CombinedOutput()
	if err == nil {
		t.Fatalf("expected stage2 test-pkg --fail-fast to fail with non-zero exit code, got success:\n%s", string(bff))
	}
	if !strings.Contains(string(bff), "\"fail_fast\":true") {
		t.Fatalf("expected fail_fast selection metadata, got:\n%s", string(bff))
	}
	if !strings.Contains(string(bff), "\"skipped\":1") {
		t.Fatalf("expected skipped count in summary for fail-fast run, got:\n%s", string(bff))
	}
	if strings.Contains(string(bff), "\"name\":\"tests::test_b_should_skip\"") {
		t.Fatalf("expected fail-fast to skip second test execution, got:\n%s", string(bff))
	}
}

func TestStage1ExitCodeNonZeroOnBuildPkgFailure(t *testing.T) {
	t.Parallel()

	// Build stage1 compiler (vox_stage1) using stage0.
	stage1Bin := stage1ToolBin(t)

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
	// Typecheck error: unknown function.
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { nope(); return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	b, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected stage1 build-pkg to fail with non-zero exit code")
	}
	if !strings.Contains(string(b), "compile failed") {
		t.Fatalf("expected compile error output, got:\n%s", string(b))
	}
	if strings.Contains(string(b), "panic") {
		t.Fatalf("expected no panic for compile error, got:\n%s", string(b))
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if ee.ExitCode() == 0 {
			t.Fatalf("expected non-zero exit code")
		}
	} else {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
}

func TestStage1BuildPkgIsQuietOnSuccess(t *testing.T) {
	t.Parallel()

	// Build stage1 compiler (vox_stage1) using stage0.
	stage1Bin := stage1ToolBin(t)

	// Create a tiny package that successfully builds; stage1 CLI should not
	// print a trailing "0" (driver return value) on success.
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
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	outBin := filepath.Join(root, "out")
	cmd := exec.Command(stage1Bin, "build-pkg", outBin)
	cmd.Dir = root
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 build-pkg failed: %v\n%s", err, string(b))
	}
	if got := strings.TrimSpace(string(b)); got != "" {
		t.Fatalf("expected no output on success, got:\n%s", got)
	}
}
