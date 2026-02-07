package stdlib

import (
	_ "embed"

	"voxlang/internal/source"
)

//go:embed src/std/prelude/lib.vox
var preludeSrc string

//go:embed src/std/testing/lib.vox
var testingSrc string

// Files returns stage0 stdlib sources that are implicitly available to all builds.
//
// These are compiled as if they were part of the root package under src/std/**.
func Files() []*source.File {
	return []*source.File{
		source.NewFile("src/std/prelude/lib.vox", preludeSrc),
		source.NewFile("src/std/testing/lib.vox", testingSrc),
	}
}
