package names

import "strings"

// PackageFromFileName returns the "package qualifier" for a source file name.
//
// Stage0 loader prefixes dependency files as: "<depName>/<relPath>".
// Root package files keep relPath like "src/main.vox" or "tests/basic.vox".
func PackageFromFileName(name string) string {
	if name == "" {
		return ""
	}
	i := strings.IndexByte(name, '/')
	if i < 0 {
		return ""
	}
	first := name[:i]
	// Root package files use src/ or tests/ as the first segment.
	if first == "src" || first == "tests" {
		return ""
	}
	return first
}

func QualifyFunc(fileName string, fnName string) string {
	pkg := PackageFromFileName(fileName)
	if pkg == "" {
		return fnName
	}
	return pkg + "::" + fnName
}
