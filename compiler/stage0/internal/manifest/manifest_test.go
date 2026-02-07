package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBasic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "vox.toml")
	if err := os.WriteFile(p, []byte(`
[package]
name = "x"
version = "0.1.0"
edition = "2026"

[dependencies]
foo = { path = "../foo" }
bar = "1.2.3"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if m.Package.Name != "x" {
		t.Fatalf("name: %q", m.Package.Name)
	}
	if m.Dependencies["foo"].Path != "../foo" {
		t.Fatalf("foo path: %q", m.Dependencies["foo"].Path)
	}
	if m.Dependencies["bar"].Version != "1.2.3" {
		t.Fatalf("bar version: %q", m.Dependencies["bar"].Version)
	}
}
