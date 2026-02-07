package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"voxlang/internal/diag"
	"voxlang/internal/interp"
	"voxlang/internal/manifest"
	"voxlang/internal/parser"
	"voxlang/internal/source"
	"voxlang/internal/stdlib"
	"voxlang/internal/typecheck"
)

type depState int

const (
	depUnvisited depState = iota
	depVisiting
	depDone
)

type BuildResult struct {
	Manifest  *manifest.Manifest
	Root      string
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
	} else {
		mani = &manifest.Manifest{
			Path:         "",
			Package:      manifest.Package{Name: filepath.Base(root), Version: "0.0.0", Edition: "2026"},
			Dependencies: map[string]manifest.Dependency{},
		}
	}

	deps, err := resolveAllPathDeps(root, mani)
	if err != nil {
		return nil, nil, err
	}

	files, err := collectPackageFiles(root, collectOptions{
		IncludeTests: tests,
		RequireMain:  !tests,
		FilePrefix:   "",
		SkipMain:     false,
	})
	if err != nil {
		return nil, nil, err
	}
	// Stage0 stdlib is always available, except when building stage1 itself:
	// stage1 owns the stdlib sources under compiler/stage1/src/std/**.
	if stage1Root, err := stdlib.Stage1RootDir(); err == nil && filepath.Clean(root) == filepath.Clean(stage1Root) {
		// no injection
	} else {
		stdFiles, err := stdlib.Files()
		if err != nil {
			return nil, nil, err
		}
		files = append(files, stdFiles...)
	}

	// Load path dependencies (including transitive).
	depNames := make([]string, 0, len(deps))
	for name := range deps {
		depNames = append(depNames, name)
	}
	sort.Strings(depNames)
	for _, depName := range depNames {
		depRoot := deps[depName]
		depFiles, err := collectPackageFiles(depRoot, collectOptions{
			IncludeTests: false,
			RequireMain:  false,
			FilePrefix:   depName + "/",
			SkipMain:     true,
		})
		if err != nil {
			return nil, nil, err
		}
		files = append(files, depFiles...)
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		return &BuildResult{Manifest: mani}, pdiags, nil
	}

	modByPkg := map[string]map[string]bool{}
	rootMods, err := collectLocalModules(root)
	if err != nil {
		return nil, nil, err
	}
	// Built-in std modules (stage0 subset).
	rootMods["std/prelude"] = true
	rootMods["std/testing"] = true
	modByPkg[""] = rootMods
	for depName, depRoot := range deps {
		mods, err := collectLocalModules(depRoot)
		if err != nil {
			return nil, nil, err
		}
		mods["std/prelude"] = true
		mods["std/testing"] = true
		modByPkg[depName] = mods
	}

	allowed := map[string]bool{}
	for name := range deps {
		allowed[name] = true
	}
	checked, tdiags := typecheck.Check(prog, typecheck.Options{AllowedPkgs: allowed, LocalModulesByPkg: modByPkg})
	if tdiags != nil && len(tdiags.Items) > 0 {
		return &BuildResult{Manifest: mani}, tdiags, nil
	}

	res := &BuildResult{Manifest: mani, Program: checked}
	res.Root = root
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
		// For stage0, dependency packages are treated as libraries and must have src/lib.vox.
		if _, err := os.Stat(filepath.Join(p, "src", "lib.vox")); err != nil {
			return fmt.Errorf("dependency %q missing src/lib.vox: %s", name, p)
		}
	}
	return nil
}

func resolveAllPathDeps(root string, mani *manifest.Manifest) (map[string]string, error) {
	// Resolve all path deps reachable from the root manifest.
	// Output maps depName -> absolute path to package root.
	resolved := map[string]string{}
	state := map[string]depState{}
	var stack []string

	var visit func(pkgDir string, name string, dep manifest.Dependency) error
	visit = func(pkgDir string, name string, dep manifest.Dependency) error {
		if dep.Path == "" {
			return nil
		}
		switch state[name] {
		case depVisiting:
			// Build cycle chain.
			i := 0
			for ; i < len(stack); i++ {
				if stack[i] == name {
					break
				}
			}
			cycle := append(append([]string{}, stack[i:]...), name)
			return fmt.Errorf("circular dependency: %s", strings.Join(cycle, " -> "))
		case depDone:
			// Ensure path matches.
			abs := dep.Path
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(pkgDir, abs)
			}
			abs, err := filepath.Abs(abs)
			if err != nil {
				return err
			}
			if prev, ok := resolved[name]; ok && prev != abs {
				return fmt.Errorf("dependency %q resolved to multiple paths: %s and %s", name, prev, abs)
			}
			return nil
		}

		state[name] = depVisiting
		stack = append(stack, name)

		abs := dep.Path
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(pkgDir, abs)
		}
		abs, err := filepath.Abs(abs)
		if err != nil {
			return err
		}
		if _, err := os.Stat(abs); err != nil {
			return fmt.Errorf("dependency %q path not found: %s", name, abs)
		}
		// Dependency packages are treated as libraries and must have src/lib.vox.
		if _, err := os.Stat(filepath.Join(abs, "src", "lib.vox")); err != nil {
			return fmt.Errorf("dependency %q missing src/lib.vox: %s", name, abs)
		}
		if prev, ok := resolved[name]; ok && prev != abs {
			return fmt.Errorf("dependency %q resolved to multiple paths: %s and %s", name, prev, abs)
		}
		resolved[name] = abs

		mp := filepath.Join(abs, "vox.toml")
		m2, err := manifest.Load(mp)
		if err != nil {
			return fmt.Errorf("load dependency %q manifest: %w", name, err)
		}
		if m2.Package.Name != name {
			return fmt.Errorf("dependency %q package name mismatch: vox.toml has name=%q", name, m2.Package.Name)
		}
		for depName, dep2 := range m2.Dependencies {
			if err := visit(abs, depName, dep2); err != nil {
				return err
			}
		}

		stack = stack[:len(stack)-1]
		state[name] = depDone
		return nil
	}

	for depName, dep := range mani.Dependencies {
		if err := visit(root, depName, dep); err != nil {
			return nil, err
		}
	}
	return resolved, nil
}

func collectLocalModules(root string) (map[string]bool, error) {
	// Module paths are directory paths under src/ that contain at least one non-test .vox file.
	srcDir := filepath.Join(root, "src")
	out := map[string]bool{}
	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == "target" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".vox" {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		// Test files are not importable modules.
		if strings.HasSuffix(rel, "_test.vox") {
			return nil
		}
		if rel == "main.vox" || rel == "lib.vox" {
			return nil
		}
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "." || dir == "" {
			// root module isn't importable
			return nil
		}
		out[dir] = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type collectOptions struct {
	IncludeTests bool
	RequireMain  bool
	FilePrefix   string
	SkipMain     bool
}

func collectPackageFiles(root string, opts collectOptions) ([]*source.File, error) {
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
			if opts.SkipMain && rel == filepath.Join("src", "main.vox") {
				return nil
			}
			// Go-style test files: only included when IncludeTests is enabled.
			if !opts.IncludeTests && strings.HasSuffix(rel, "_test.vox") {
				return nil
			}
			out = append(out, source.NewFile(opts.FilePrefix+rel, string(b)))
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
	if opts.IncludeTests {
		testDir := filepath.Join(root, "tests")
		if _, err := os.Stat(testDir); err == nil {
			if err := addDir(testDir); err != nil {
				return nil, err
			}
		}
	}
	// Stage0 expects main for executable builds.
	if opts.RequireMain && !opts.SkipMain {
		mainPath := filepath.Join(srcDir, "main.vox")
		if _, err := os.Stat(mainPath); err != nil {
			return nil, fmt.Errorf("missing src/main.vox in %s", root)
		}
	}
	return out, nil
}
