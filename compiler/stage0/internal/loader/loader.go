package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"voxlang/internal/diag"
	"voxlang/internal/interp"
	"voxlang/internal/manifest"
	"voxlang/internal/parser"
	"voxlang/internal/source"
	"voxlang/internal/typecheck"
)

type BuildResult struct {
	Manifest  *manifest.Manifest
	Program   *typecheck.CheckedProgram
	RunResult string
	TestLog   string
}

func InitPackage(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(abs, "src"), 0o755); err != nil {
		return err
	}
	name := filepath.Base(abs)

	manifestPath := filepath.Join(abs, "vox.toml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		content := fmt.Sprintf(`[package]
name = %q
version = "0.1.0"
edition = "2026"

[dependencies]
`, name)
		if err := os.WriteFile(manifestPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	mainPath := filepath.Join(abs, "src", "main.vox")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		content := `fn main() -> i32 {
  return 0;
}
`
		if err := os.WriteFile(mainPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func BuildPackage(dir string, run bool) (*BuildResult, *diag.Bag, error) {
	return buildPackage(dir, run, false)
}

func TestPackage(dir string) (*BuildResult, *diag.Bag, error) {
	return buildPackage(dir, false, true)
}

func buildPackage(dir string, run bool, tests bool) (*BuildResult, *diag.Bag, error) {
	root, maniPath, err := findPackageRoot(dir)
	if err != nil {
		return nil, nil, err
	}
	var mani *manifest.Manifest
	if maniPath != "" {
		mani, err = manifest.Load(maniPath)
		if err != nil {
			return nil, nil, err
		}
		if err := validateDeps(root, mani); err != nil {
			return nil, nil, err
		}
	} else {
		mani = &manifest.Manifest{
			Path:         "",
			Package:      manifest.Package{Name: filepath.Base(root), Version: "0.0.0", Edition: "2026"},
			Dependencies: map[string]manifest.Dependency{},
		}
	}

	files, err := collectPackageFiles(root, tests)
	if err != nil {
		return nil, nil, err
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		return &BuildResult{Manifest: mani}, pdiags, nil
	}
	checked, tdiags := typecheck.Check(prog)
	if tdiags != nil && len(tdiags.Items) > 0 {
		return &BuildResult{Manifest: mani}, tdiags, nil
	}

	res := &BuildResult{Manifest: mani, Program: checked}
	if tests {
		log, terr := interp.RunTests(checked)
		res.TestLog = log
		if terr != nil {
			db := &diag.Bag{}
			db.Add(root, 1, 1, terr.Error())
			return res, db, nil
		}
	}
	if run {
		out, rerr := interp.RunMain(checked)
		if rerr != nil {
			db := &diag.Bag{}
			db.Add(root, 1, 1, rerr.Error())
			return res, db, nil
		}
		res.RunResult = out
	}
	return res, nil, nil
}

func findPackageRoot(dir string) (root string, manifestPath string, err error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	cur := abs
	for {
		mp := filepath.Join(cur, "vox.toml")
		if _, err := os.Stat(mp); err == nil {
			return cur, mp, nil
		}
		// fallback: directory with src/main.vox
		if _, err := os.Stat(filepath.Join(cur, "src", "main.vox")); err == nil {
			return cur, "", nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return abs, "", nil
}

func validateDeps(root string, mani *manifest.Manifest) error {
	for name, dep := range mani.Dependencies {
		if dep.Path == "" {
			// registry deps are deferred
			continue
		}
		p := dep.Path
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("dependency %q path not found: %s", name, p)
		}
		// If the dependency is a Vox package, it should have vox.toml (soft requirement for now).
	}
	return nil
}

func collectPackageFiles(root string, includeTests bool) ([]*source.File, error) {
	var out []*source.File
	addDir := func(dir string) error {
		return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				// skip target-style dirs if present
				base := filepath.Base(path)
				if base == "target" || strings.HasPrefix(base, ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".vox" {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel := strings.TrimPrefix(path, root+string(filepath.Separator))
			out = append(out, source.NewFile(rel, string(b)))
			return nil
		})
	}
	srcDir := filepath.Join(root, "src")
	if _, err := os.Stat(srcDir); err != nil {
		return nil, fmt.Errorf("missing src/ in %s", root)
	}
	if err := addDir(srcDir); err != nil {
		return nil, err
	}
	if includeTests {
		testDir := filepath.Join(root, "tests")
		if _, err := os.Stat(testDir); err == nil {
			if err := addDir(testDir); err != nil {
				return nil, err
			}
		}
	}
	// Stage0 expects main when running (not for tests-only).
	if !includeTests {
		mainPath := filepath.Join(srcDir, "main.vox")
		if _, err := os.Stat(mainPath); err != nil {
			return nil, fmt.Errorf("missing src/main.vox in %s", root)
		}
	}
	return out, nil
}
