package names

import (
	"path/filepath"
	"sort"
	"strings"
)

type TestModuleGroup struct {
	Key   string
	Tests []string
}

// SplitOwnerAndModule derives the "owning package" and module path from a source file name.
//
// Stage0 loader prefixes dependency files as: "<depName>/<relPath>".
// Root package files keep relPath like "src/main.vox" or "tests/basic.vox".
//
// Module path rules (stage0):
//   - src/main.vox is the executable entrypoint but still belongs to the root module.
//   - Any file directly under src/ belongs to the root module (file name doesn't affect module path).
//   - src/**.vox files belong to the module represented by their directory path under src/.
//     Examples:
//   - src/a/a.vox, src/a/x.vox              -> module ["a"]
//   - src/utils/io/file.vox, src/utils/io/x.vox -> module ["utils","io"]
//   - src/**/*_test.vox are treated as part of the directory module (file name doesn't form a module segment).
//   - tests/**.vox are treated as belonging to a separate top-level module "tests" (and its subdirectories),
//     so they cannot access private symbols from src/** by default (Go-like "external tests" behavior).
func SplitOwnerAndModule(fileName string) (pkg string, mod []string, isTest bool) {
	rel := filepath.ToSlash(fileName)
	if rel == "" {
		return "", nil, false
	}

	// Optional dependency prefix: "<depName>/..."
	first, rest, ok := strings.Cut(rel, "/")
	if ok && first != "src" && first != "tests" {
		// Namespace dependency packages to avoid collisions with local modules.
		// For example:
		// - root local module: src/dep/** -> qname "dep::foo"
		// - dependency package: dep/src/** -> qname "pkg.dep::foo"
		pkg = "pkg." + first
		rel = rest
	}

	if rel == "tests" {
		return pkg, []string{"tests"}, true
	}
	if strings.HasPrefix(rel, "tests/") {
		isTest = true
		path := strings.TrimPrefix(rel, "tests/")
		dir := filepath.ToSlash(filepath.Dir(path))
		out := []string{"tests"}
		if dir != "." && dir != "" {
			segs := strings.Split(dir, "/")
			for _, s := range segs {
				if s == "" || s == "." {
					continue
				}
				out = append(out, s)
			}
		}
		return pkg, out, true
	}
	if !strings.HasPrefix(rel, "src/") {
		// Unknown layout; treat as root module.
		return pkg, nil, false
	}

	path := strings.TrimPrefix(rel, "src/")
	if strings.HasSuffix(path, "_test.vox") {
		isTest = true
	}

	// Directory-as-module: module path is the directory segments under src/.
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." || dir == "" {
		return pkg, nil, isTest
	}
	segs := strings.Split(dir, "/")
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if s == "" || s == "." {
			continue
		}
		out = append(out, s)
	}
	return pkg, out, isTest
}

func QualifyParts(pkg string, mod []string, fnName string) string {
	var b strings.Builder
	if pkg != "" {
		b.WriteString(pkg)
		b.WriteString("::")
	}
	if len(mod) > 0 {
		b.WriteString(strings.Join(mod, "."))
		b.WriteString("::")
	}
	b.WriteString(fnName)
	return b.String()
}

func QualifyFunc(fileName string, fnName string) string {
	pkg, mod, _ := SplitOwnerAndModule(fileName)
	return QualifyParts(pkg, mod, fnName)
}

// PackageFromFileName returns the dependency package name prefix (if any).
func PackageFromFileName(name string) string {
	pkg, _, _ := SplitOwnerAndModule(name)
	return pkg
}

// GroupQualifiedTestsByModule groups qualified test function names by module key.
//
// Input names should already be sorted if deterministic per-module order is desired.
func GroupQualifiedTestsByModule(testNames []string) []TestModuleGroup {
	if len(testNames) == 0 {
		return nil
	}
	groupMap := map[string][]string{}
	keys := make([]string, 0)
	for _, name := range testNames {
		key := ""
		if i := strings.LastIndex(name, "::"); i >= 0 {
			key = name[:i]
		}
		if _, ok := groupMap[key]; !ok {
			keys = append(keys, key)
		}
		groupMap[key] = append(groupMap[key], name)
	}
	sort.Strings(keys)
	out := make([]TestModuleGroup, 0, len(keys))
	for _, key := range keys {
		out = append(out, TestModuleGroup{Key: key, Tests: groupMap[key]})
	}
	return out
}
