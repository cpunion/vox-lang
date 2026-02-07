package manifest

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Manifest struct {
	Path         string
	Package      Package
	Dependencies map[string]Dependency
}

type Package struct {
	Name    string
	Version string
	Edition string
}

type Dependency struct {
	Version string
	Path    string
}

func Load(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &Manifest{Path: path, Dependencies: map[string]Dependency{}}
	var section string
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, val, ok := cutKV(line)
		if !ok {
			return nil, fmt.Errorf("%s: invalid line: %q", path, line)
		}
		switch section {
		case "package":
			switch key {
			case "name":
				m.Package.Name = unquote(val)
			case "version":
				m.Package.Version = unquote(val)
			case "edition":
				m.Package.Edition = unquote(val)
			}
		case "dependencies":
			dep := Dependency{}
			if strings.HasPrefix(val, "{") {
				// minimal inline table: { path = "..." }
				dep.Path = parseInlinePath(val)
			} else {
				dep.Version = unquote(val)
			}
			m.Dependencies[key] = dep
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if m.Package.Name == "" {
		// default to directory name
		m.Package.Name = filepath.Base(filepath.Dir(path))
	}
	return m, nil
}

func cutKV(line string) (key, val string, ok bool) {
	i := strings.IndexByte(line, '=')
	if i < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:i])
	val = strings.TrimSpace(line[i+1:])
	if key == "" || val == "" {
		return "", "", false
	}
	return key, val, true
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func parseInlinePath(val string) string {
	// super small subset: { path = "..." }
	val = strings.TrimSpace(val)
	if !strings.HasPrefix(val, "{") || !strings.HasSuffix(val, "}") {
		return ""
	}
	inner := strings.TrimSpace(val[1 : len(val)-1])
	parts := strings.Split(inner, ",")
	for _, p := range parts {
		k, v, ok := cutKV(strings.TrimSpace(p))
		if !ok {
			continue
		}
		if k == "path" {
			return unquote(v)
		}
	}
	return ""
}
