package stdlib

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"voxlang/internal/source"
)

// Files returns stage0 stdlib sources that are implicitly available to all builds.
//
// These are compiled as if they were part of the root package under src/std/**.
func Files() ([]*source.File, error) {
	loadOnce.Do(load)
	if loadErr != nil {
		return nil, loadErr
	}
	out := make([]*source.File, len(loaded))
	copy(out, loaded)
	return out, nil
}

func Stage1RootDir() (string, error) {
	root, err := stage0RootDir()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(root, "..", "stage1")), nil
}

var (
	loadOnce sync.Once
	loadErr  error
	loaded   []*source.File
)

func load() {
	root, err := stage0RootDir()
	if err != nil {
		loadErr = err
		return
	}
	// Single source of truth: stage1 stdlib sources are written in Vox and live under
	// compiler/stage1/src/std/** so stage1 can directly ship/use them.
	stage1Root := filepath.Clean(filepath.Join(root, "..", "stage1"))
	stdRoot := filepath.Join(stage1Root, "src", "std")
	files, err := readStdlibDir(stdRoot)
	if err != nil {
		loadErr = err
		return
	}
	loaded = files
}

func stage0RootDir() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok || file == "" {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	// stdlib.go lives at: <repo>/compiler/stage0/internal/stdlib/stdlib.go
	// stage0 root is:      <repo>/compiler/stage0
	dir := filepath.Dir(file)
	return filepath.Clean(filepath.Join(dir, "..", "..")), nil
}

func readStdlibDir(stdRoot string) ([]*source.File, error) {
	var paths []string
	err := filepath.WalkDir(stdRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".vox" {
			return nil
		}
		rel, err := filepath.Rel(stdRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		// Don't inject stdlib tests as part of every build.
		if strings.HasSuffix(rel, "_test.vox") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read stdlib dir %q: %w", stdRoot, err)
	}
	sort.Strings(paths)
	out := make([]*source.File, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read stdlib file %q: %w", p, err)
		}
		rel, err := filepath.Rel(stdRoot, p)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		out = append(out, source.NewFile("src/std/"+rel, string(b)))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("stdlib directory is empty: %s", stdRoot)
	}
	return out, nil
}
