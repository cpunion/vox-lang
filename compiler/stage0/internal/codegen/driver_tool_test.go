package codegen

import (
	"os/exec"
	"path/filepath"
	"testing"

	"voxlang/internal/irgen"
	"voxlang/internal/source"
)

func TestToolDriverMainIsQuietAndReturnsExitCode(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	checked := parseAndCheckWithStdlib(t, []*source.File{
		source.NewFile("main.vox", `fn main() -> i32 { return 7; }`),
	})
	irp, err := irgen.Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(irp, EmitOptions{EmitDriverMain: true, DriverMainKind: DriverMainTool})
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
		t.Fatalf("expected non-zero exit; got success with output %q", string(out))
	}
	if got := string(out); got != "" {
		t.Fatalf("expected no output, got %q", got)
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if ee.ExitCode() != 7 {
			t.Fatalf("expected exit code 7, got %d", ee.ExitCode())
		}
		return
	}
	t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
}
