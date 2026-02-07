package loader

import (
	"os"
	"path/filepath"
	"strings"
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

func TestBuildPackage_ImportsAreFileLocal(t *testing.T) {
	dir := t.TempDir()
	// dep package
	depDir := filepath.Join(dir, "dep")
	mustWrite(t, filepath.Join(depDir, "vox.toml"), `[package]
name = "dep"
version = "0.1.0"
edition = "2026"

[dependencies]
`)
	mustWrite(t, filepath.Join(depDir, "src", "lib.vox"), `fn one() -> i32 { return 1; }`)

	// root package
	rootDir := filepath.Join(dir, "app")
	mustWrite(t, filepath.Join(rootDir, "vox.toml"), `[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "../dep" }
`)
	// import in a.vox only
	mustWrite(t, filepath.Join(rootDir, "src", "a.vox"), `import "dep"
fn ok() -> i32 { return dep.one(); }`)
	// main tries to use dep without importing it
	mustWrite(t, filepath.Join(rootDir, "src", "main.vox"), `fn main() -> i32 { return dep.one(); }`)

	_, diags, err := BuildPackage(rootDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if diags == nil || len(diags.Items) == 0 {
		t.Fatalf("expected diagnostics")
	}
	found := false
	for _, it := range diags.Items {
		if strings.Contains(it.Msg, "did you forget `import \"dep\"`") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing import diag, got: %+v", diags.Items)
	}
}

func TestBuildPackage_DuplicateImportAliasInFile(t *testing.T) {
	dir := t.TempDir()
	// dep package
	depDir := filepath.Join(dir, "dep")
	mustWrite(t, filepath.Join(depDir, "vox.toml"), `[package]
name = "dep"
version = "0.1.0"
edition = "2026"

[dependencies]
`)
	mustWrite(t, filepath.Join(depDir, "src", "lib.vox"), `fn one() -> i32 { return 1; }`)

	// root package
	rootDir := filepath.Join(dir, "app")
	mustWrite(t, filepath.Join(rootDir, "vox.toml"), `[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
dep = { path = "../dep" }
`)
	mustWrite(t, filepath.Join(rootDir, "src", "main.vox"), `import "dep" as d
import "dep" as d
fn main() -> i32 { return d.one(); }`)

	_, diags, err := BuildPackage(rootDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if diags == nil || len(diags.Items) == 0 {
		t.Fatalf("expected diagnostics")
	}
	found := false
	for _, it := range diags.Items {
		if it.Msg == "duplicate import alias: d" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected duplicate import alias diag, got: %+v", diags.Items)
	}
}

func TestBuildPackage_TransitivePathDepsAreLoadable(t *testing.T) {
	dir := t.TempDir()

	// b
	bDir := filepath.Join(dir, "b")
	mustWrite(t, filepath.Join(bDir, "vox.toml"), `[package]
name = "b"
version = "0.1.0"
edition = "2026"

[dependencies]
`)
	mustWrite(t, filepath.Join(bDir, "src", "lib.vox"), `fn one() -> i32 { return 1; }`)

	// a depends on b
	aDir := filepath.Join(dir, "a")
	mustWrite(t, filepath.Join(aDir, "vox.toml"), `[package]
name = "a"
version = "0.1.0"
edition = "2026"

[dependencies]
b = { path = "../b" }
`)
	mustWrite(t, filepath.Join(aDir, "src", "lib.vox"), `import "b"
fn a_one() -> i32 { return b.one(); }`)

	// app depends on a; app imports b directly (transitive import)
	appDir := filepath.Join(dir, "app")
	mustWrite(t, filepath.Join(appDir, "vox.toml"), `[package]
name = "app"
version = "0.1.0"
edition = "2026"

[dependencies]
a = { path = "../a" }
`)
	mustWrite(t, filepath.Join(appDir, "src", "main.vox"), `import "b"
fn main() -> i32 { return b.one(); }`)

	_, diags, err := BuildPackage(appDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags.Items)
	}
}

func TestBuildPackage_DependencyCycleIsError(t *testing.T) {
	dir := t.TempDir()

	aDir := filepath.Join(dir, "a")
	bDir := filepath.Join(dir, "b")

	mustWrite(t, filepath.Join(aDir, "vox.toml"), `[package]
name = "a"
version = "0.1.0"
edition = "2026"

[dependencies]
b = { path = "../b" }
`)
	mustWrite(t, filepath.Join(aDir, "src", "main.vox"), `fn main() -> i32 { return 0; }`)
	mustWrite(t, filepath.Join(aDir, "src", "lib.vox"), `fn a() -> i32 { return 0; }`)

	mustWrite(t, filepath.Join(bDir, "vox.toml"), `[package]
name = "b"
version = "0.1.0"
edition = "2026"

[dependencies]
a = { path = "../a" }
`)
	mustWrite(t, filepath.Join(bDir, "src", "lib.vox"), `fn b() -> i32 { return 0; }`)

	_, _, err := BuildPackage(aDir, false)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "circular dependency:") {
		t.Fatalf("expected circular dependency error, got %v", err)
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
