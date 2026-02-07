package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPackage_UnknownLocalModuleImport(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "vox.toml"), `[package]
name = "a"
version = "0.1.0"
edition = "2026"

[dependencies]
`)
	mustWrite(t, filepath.Join(dir, "src", "main.vox"), `import "nope"
fn main() -> i32 { return 0; }`)

	_, diags, err := BuildPackage(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if diags == nil || len(diags.Items) == 0 {
		t.Fatalf("expected diagnostics")
	}
	found := false
	for _, it := range diags.Items {
		if it.Msg == "unknown local module: nope" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unknown local module diag, got: %+v", diags.Items)
	}
}

func TestBuildPackage_LocalModuleImportResolves(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "vox.toml"), `[package]
name = "a"
version = "0.1.0"
edition = "2026"

[dependencies]
`)
	mustWrite(t, filepath.Join(dir, "src", "main.vox"), `import "utils"
fn main() -> i32 { return utils.one(); }`)
	mustWrite(t, filepath.Join(dir, "src", "utils", "lib.vox"), `fn one() -> i32 { return 1; }`)

	_, diags, err := BuildPackage(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags.Items)
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
