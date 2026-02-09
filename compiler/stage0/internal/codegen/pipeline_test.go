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
		source.NewFile("dep/src/dep.vox", `pub fn one() -> i32 { return 1; }`),
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

func TestPipelineIfExprAndVecFieldPushCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `struct S { items: Vec[i32] }
fn main() -> i32 {
  let mut s: S = S { items: Vec() };
  let x: i32 = if true { 40 } else { 0 };
  s.items.push(x);
  s.items.push(2);
  return s.items.get(0) + s.items.get(1) + s.items.len();
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

func TestPipelineEnumEqualityAgainstUnitVariantCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `enum E { A(i32), None }
fn main() -> i32 {
  let x: E = E.A(1);
  if x == E.None { return 0; } else { return 1; }
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

func TestPipelineVecMemberCallsCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `struct S { v: Vec[i32] }
fn main() -> i32 {
  let mut v: Vec[i32] = Vec();
  v.push(2);
  v.push(3);
  let s: S = S { v: v };
  return s.v.len() + s.v.get(0) + s.v.get(1);
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
	if got := strings.TrimSpace(string(out)); got != "7" {
		t.Fatalf("expected output 7, got %q", got)
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

func TestPipelineEnumMultiPayloadCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `enum E { Pair(i32, i32), None }
fn main() -> i32 {
  let x: E = E.Pair(40, 2);
  return match x {
    E.Pair(a, b) => a + b,
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

func TestPipelineStringLenAndByteAtCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let s: String = "abc";
  let n: i32 = s.len();
  let b0: i32 = s.byte_at(0);
  return n + b0;
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
	if got := strings.TrimSpace(string(out)); got != "100" {
		t.Fatalf("expected output 100, got %q", got)
	}
}

func TestPipelineStringSliceCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let s: String = "abc";
  let t: String = s.slice(1, 3); // "bc"
  return t.len() + t.byte_at(0); // 2 + 'b'(98) = 100
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
	if got := strings.TrimSpace(string(out)); got != "100" {
		t.Fatalf("expected output 100, got %q", got)
	}
}

func TestPipelineStringConcatCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let a: String = "ab";
  let b: String = "cd";
  let c: String = a.concat(b); // "abcd"
  return c.len() + c.byte_at(0); // 4 + 'a'(97) = 101
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
	if got := strings.TrimSpace(string(out)); got != "101" {
		t.Fatalf("expected output 101, got %q", got)
	}
}

func TestPipelineVecStringJoinCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let mut v: Vec[String] = Vec();
  v.push("a");
  v.push("b");
  let s: String = v.join(",");
  return s.len() + s.byte_at(0); // "a,b" => 3 + 'a'(97) = 100
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
	if got := strings.TrimSpace(string(out)); got != "100" {
		t.Fatalf("expected output 100, got %q", got)
	}
}

func TestPipelineEscapeCAndToStringCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let s: String = "a\nb";
  let e: String = s.escape_c(); // "a\\nb"
  let n: i32 = 42;
  let x: String = n.to_string(); // "42"
  let u: u64 = 42;
  let y: String = u.to_string(); // "42"
  // e: len 4 + byte_at(1) '\\'(92) = 96
  // x: len 2 + byte_at(0) '4'(52) = 54
  // y: len 2 + byte_at(0) '4'(52) = 54
  return e.len() + e.byte_at(1) + x.len() + x.byte_at(0) + y.len() + y.byte_at(0);
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
	if got := strings.TrimSpace(string(out)); got != "204" {
		t.Fatalf("expected output 204, got %q", got)
	}
}

func TestPipelineAsCastI64ToI32CheckedOverflowPanics(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let x: i64 = 3000000000;
  let y: i32 = x as i32;
  return y;
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
	if err == nil {
		t.Fatalf("expected run to fail, got output:\n%s", string(out))
	}
	if !strings.Contains(string(out), "i64 to i32 overflow") {
		t.Fatalf("expected overflow message, got:\n%s", string(out))
	}
}

func TestPipelineNegativeI8LiteralFoldsToConst(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let x: i8 = -5;
  return x as i32;
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
	if !strings.Contains(csrc, "INT8_C(-5)") {
		t.Fatalf("expected folded negative literal, got C:\n%s", csrc)
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
	if got := strings.TrimSpace(string(out)); got != "-5" {
		t.Fatalf("expected output -5, got %q", got)
	}
}

func TestPipelineNegativeI16LiteralFoldsToConst(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let x: i16 = -5;
  return x as i32;
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
	if !strings.Contains(csrc, "INT16_C(-5)") {
		t.Fatalf("expected folded negative literal, got C:\n%s", csrc)
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
	if got := strings.TrimSpace(string(out)); got != "-5" {
		t.Fatalf("expected output -5, got %q", got)
	}
}

func TestPipelineI32AddWrapsInCBackend(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let x: i32 = 2147483647;
  let y: i32 = x + 1;
  return y;
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
	if got := strings.TrimSpace(string(out)); got != "-2147483648" {
		t.Fatalf("expected output -2147483648, got %q\nC:\n%s", got, csrc)
	}
}

func TestPipelineDivByZeroPanicsInCBackend(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `fn main() -> i32 {
  let x: i32 = 1;
  let y: i32 = x / 0;
  return y;
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
	if err == nil {
		t.Fatalf("expected run to fail, got output:\n%s", string(out))
	}
	if !strings.Contains(string(out), "division by zero") {
		t.Fatalf("expected division by zero message, got:\n%s", string(out))
	}
}

func TestPipelineRangeCastPanics(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("src/main.vox", `type Tiny = @range(0..=3) i32
fn main() -> i32 {
  let x: i32 = 5;
  let y: Tiny = x as Tiny;
  return y;
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
	if err == nil {
		t.Fatalf("expected run to fail, got output:\n%s", string(out))
	}
	if !strings.Contains(string(out), "range check failed") {
		t.Fatalf("expected range error message, got:\n%s", string(out))
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
		source.NewFile("dep/src/dep.vox", `pub fn one() -> i32 { return 10; }
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
		source.NewFile("src/a/a.vox", `pub fn b() -> i32 { return 2; }`),
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
