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
	"voxlang/internal/typecheck"
)

func TestPipelineCompilesAndRuns(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	f := source.NewFile("main.vox", `fn main() -> i32 { return 0; }`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog)
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
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
		source.NewFile("src/main.vox", `fn main() -> i32 { return dep::one(); }`),
		source.NewFile("dep/src/lib.vox", `fn one() -> i32 { return 1; }`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog)
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
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

func writeFile(path string, s string) error {
	// Keep helper local to avoid pulling additional deps into stage0.
	return os.WriteFile(path, []byte(s), 0o644)
}
