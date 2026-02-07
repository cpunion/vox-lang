package names

import (
	"path/filepath"
	"strings"
)

// SplitOwnerAndModule derives the "owning package" and module path from a source file name.
//
// Stage0 loader prefixes dependency files as: "<depName>/<relPath>".
// Root package files keep relPath like "src/main.vox" or "tests/basic.vox".
//
// Module path rules (stage0):
// - src/main.vox and src/lib.vox are the root module (empty module path).
// - src/foo.vox -> module ["foo"]
// - src/utils/lib.vox -> module ["utils"]
// - src/utils/io.vox -> module ["utils","io"]
// - src/**/*_test.vox are treated as part of the directory module (file name doesn't form a module segment).
// - tests/**.vox are treated as root module for now (empty module path), to keep test discovery simple.
func SplitOwnerAndModule(fileName string) (pkg string, mod []string, isTest bool) {
	rel := filepath.ToSlash(fileName)
	if rel == "" {
		return "", nil, false
	}

	// Optional dependency prefix: "<depName>/..."
	first, rest, ok := strings.Cut(rel, "/")
	if ok && first != "src" && first != "tests" {
		pkg = first
		rel = rest
	}

	if strings.HasPrefix(rel, "tests/") || rel == "tests" {
		return pkg, nil, true
	}
	if !strings.HasPrefix(rel, "src/") {
		// Unknown layout; treat as root module.
		return pkg, nil, false
	}

	path := strings.TrimPrefix(rel, "src/")
	// Go-like test files live "in the package/module" and should not introduce an extra module segment.
	// Example: src/stage1_tests/smoke_test.vox => module ["stage1_tests"] (not ["stage1_tests","smoke_test"]).
	if strings.HasSuffix(path, "_test.vox") {
		isTest = true
		dir := filepath.ToSlash(filepath.Dir(path))
		if dir == "." || dir == "" {
			return pkg, nil, true
		}
		segs := strings.Split(dir, "/")
		out := make([]string, 0, len(segs))
		for _, s := range segs {
			if s == "" || s == "." {
				continue
			}
			out = append(out, s)
		}
		return pkg, out, true
	}
	path = strings.TrimSuffix(path, ".vox")
	if path == "main" || path == "lib" {
		return pkg, nil, false
	}
	if strings.HasSuffix(path, "/lib") {
		path = strings.TrimSuffix(path, "/lib")
	}
	segs := strings.Split(path, "/")
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if s == "" || s == "." {
			continue
		}
		out = append(out, s)
	}
	return pkg, out, false
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
