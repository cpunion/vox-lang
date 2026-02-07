package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"voxlang/internal/irgen"
	"voxlang/internal/parser"
	"voxlang/internal/source"
	"voxlang/internal/stdlib"
	"voxlang/internal/typecheck"
)

func parseAndCheckWithStdlib(t *testing.T, files []*source.File) *typecheck.CheckedProgram {
	t.Helper()
	stdFiles, err := stdlib.Files()
	if err != nil {
		t.Fatal(err)
	}
	all := append([]*source.File{}, stdFiles...)
	all = append(all, files...)
	prog, pdiags := parser.ParseFiles(all)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	return checked
}

func TestPipelineCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("main.vox", `fn main() -> i32 { return 0; }`),
	})
	irp, err := irgen.Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(irp, EmitOptions{EmitDriverMain: true})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cPath := filepath.Join(dir, "a.c")
	binPath := filepath.Join(dir, "a.out")
	if err := writeFile(cPath, csrc); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, string(out))
	}
	run := exec.Command(binPath)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "0" {
		t.Fatalf("expected output 0, got %q", got)
	}
}

func TestPipelineCompilesAndRunsWithDepQualifiedNames(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	files := []*source.File{
		source.NewFile("src/main.vox", `import "dep"
fn main() -> i32 { return dep.one(); }`),
		source.NewFile("dep/src/lib.vox", `pub fn one() -> i32 { return 1; }`),
	}
	checked := parseAndCheckWithStdlib(t, files)
	irp, err := irgen.Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(irp, EmitOptions{EmitDriverMain: true})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cPath := filepath.Join(dir, "a.c")
	binPath := filepath.Join(dir, "a.out")
	if err := writeFile(cPath, csrc); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, string(out))
	}
	run := exec.Command(binPath)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "1" {
		t.Fatalf("expected output 1, got %q", got)
	}
}

func TestPipelineStructCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `struct Point { x: i32, y: i32 }
fn main() -> i32 {
  let mut p: Point = Point { x: 1, y: 2 };
  let a: i32 = p.x;
  p.x = a + 1;
  return p.x + p.y;
}`),
	})
	irp, err := irgen.Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(irp, EmitOptions{EmitDriverMain: true})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cPath := filepath.Join(dir, "a.c")
	binPath := filepath.Join(dir, "a.out")
	if err := writeFile(cPath, csrc); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, string(out))
	}
	run := exec.Command(binPath)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "4" {
		t.Fatalf("expected output 4, got %q", got)
	}
}

func TestPipelineGenericFuncCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn id[T](x: T) -> T { return x; }
fn main() -> i32 { return id(41) + 1; }`),
	})
	irp, err := irgen.Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(irp, EmitOptions{EmitDriverMain: true})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cPath := filepath.Join(dir, "a.c")
	binPath := filepath.Join(dir, "a.out")
	if err := writeFile(cPath, csrc); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, string(out))
	}
	run := exec.Command(binPath)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "42" {
		t.Fatalf("expected output 42, got %q", got)
	}
}

func TestPipelineVecCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let mut v: Vec[i32] = Vec();
  v.push(41);
  v.push(1);
  return v.get(0) + v.get(1) + v.len();
}`),
	})
	irp, err := irgen.Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(irp, EmitOptions{EmitDriverMain: true})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cPath := filepath.Join(dir, "a.c")
	binPath := filepath.Join(dir, "a.out")
	if err := writeFile(cPath, csrc); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, string(out))
	}
	run := exec.Command(binPath)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "44" {
		t.Fatalf("expected output 44, got %q", got)
	}
}

func TestPipelineEnumCtorAndMatchCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `enum E { A(i32), B(String), None }
fn main() -> i32 {
  // enum constructor call + match expression (payload types differ across variants)
  let x: E = E.B("hi");
  let ok: bool = match x {
    E.A(v) => v == 0,
    E.B(s) => s == "hi",
    E.None => false,
  };
  assert(ok);

  let y: E = E.A(41);
  return match y {
    E.A(v) => v + 1,
    E.B(s) => 0,
    E.None => 0,
  };
}`),
	})
	irp, err := irgen.Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(irp, EmitOptions{EmitDriverMain: true})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cPath := filepath.Join(dir, "a.c")
	binPath := filepath.Join(dir, "a.out")
	if err := writeFile(cPath, csrc); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, string(out))
	}
	run := exec.Command(binPath)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "42" {
		t.Fatalf("expected output 42, got %q", got)
	}
}

func TestPipelineStructFieldOfEnumTypeCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `enum K { A, B }
struct S { k: K }
fn main() -> i32 {
  let s: S = S { k: K.A };
  let k: K = s.k;
  return match k { K.A => 1, K.B => 2 };
}`),
	})
	irp, err := irgen.Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(irp, EmitOptions{EmitDriverMain: true})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cPath := filepath.Join(dir, "a.c")
	binPath := filepath.Join(dir, "a.out")
	if err := writeFile(cPath, csrc); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, string(out))
	}
	run := exec.Command(binPath)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "1" {
		t.Fatalf("expected output 1, got %q", got)
	}
}

func TestPipelineNamedImportsForTypesAndEnumsCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	files := []*source.File{
		source.NewFile("src/main.vox", `import { S, E, one } from "dep"
fn main() -> i32 {
  let s: S = S { x: 1 };
  let e: E = E.A(41);
  let v: i32 = match e {
    E.A(n) => n + 1,
    E.None => 0,
  };
  return one() + s.x + v;
}`),
		source.NewFile("dep/src/lib.vox", `pub fn one() -> i32 { return 10; }
pub struct S { pub x: i32 }
pub enum E { A(i32), None }
`),
	}
	checked := parseAndCheckWithStdlib(t, files)
	irp, err := irgen.Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(irp, EmitOptions{EmitDriverMain: true})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cPath := filepath.Join(dir, "a.c")
	binPath := filepath.Join(dir, "a.out")
	if err := writeFile(cPath, csrc); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, string(out))
	}
	run := exec.Command(binPath)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(out))
	}
	// one()=10, s.x=1, match(E.A(41))=42 => 53
	if got := strings.TrimSpace(string(out)); got != "53" {
		t.Fatalf("expected output 53, got %q", got)
	}
}

func TestPipelineNoSymbolCollisionBetweenQualifiedAndPlainNames(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	// Ensure C backend name mangling is collision-free across modules/packages.
	// This used to collide:
	// - root function: a__b
	// - module a function: b  => qualified name "a::b"
	files := []*source.File{
		source.NewFile("src/main.vox", `import "a" as a
fn a__b() -> i32 { return 1; }
fn main() -> i32 {
  return a__b() + a.b();
}`),
		source.NewFile("src/a.vox", `pub fn b() -> i32 { return 2; }`),
	}
	checked := parseAndCheckWithStdlib(t, files)
	irp, err := irgen.Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(irp, EmitOptions{EmitDriverMain: true})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cPath := filepath.Join(dir, "a.c")
	binPath := filepath.Join(dir, "a.out")
	if err := writeFile(cPath, csrc); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, string(out))
	}
	run := exec.Command(binPath)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "3" {
		t.Fatalf("expected output 3, got %q", got)
	}
}

func writeFile(path string, s string) error {
	// Keep helper local to avoid pulling additional deps into stage0.
	return os.WriteFile(path, []byte(s), 0o644)
}
