package stdlib

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	preludePath := filepath.Join(stage1Root, "src", "std", "prelude", "lib.vox")
	testingPath := filepath.Join(stage1Root, "src", "std", "testing", "lib.vox")

	preludeSrc, err := os.ReadFile(preludePath)
	if err != nil {
		loadErr = fmt.Errorf("read stdlib file %q: %w", preludePath, err)
		return
	}
	testingSrc, err := os.ReadFile(testingPath)
	if err != nil {
		loadErr = fmt.Errorf("read stdlib file %q: %w", testingPath, err)
		return
	}

	loaded = []*source.File{
		// Note: file names are virtualized to keep module resolution stable.
		source.NewFile("src/std/prelude/lib.vox", string(preludeSrc)),
		source.NewFile("src/std/testing/lib.vox", string(testingSrc)),
	}
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
