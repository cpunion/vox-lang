package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitCForDir_Smoke(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vox.toml"), []byte("[package]\nname = \"c_emit_smoke\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := "fn main() -> i32 { return 0; }\n"
	if err := os.WriteFile(filepath.Join(dir, "src", "main.vox"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	csrc, err := emitCForDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure this is a complete, compilable translation unit with a driver main.
	if !strings.Contains(csrc, "int main(int argc, char** argv)") {
		t.Fatalf("expected driver main in generated C, got:\n%s", csrc[:min(2000, len(csrc))])
	}
	if !strings.Contains(csrc, "vox_fn_mmain") {
		t.Fatalf("expected user main symbol in generated C, got:\n%s", csrc[:min(2000, len(csrc))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
