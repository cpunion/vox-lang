package stdlib

import "voxlang/internal/source"

// Files returns stage0 stdlib sources that are implicitly available to all builds.
//
// These are compiled as if they were part of the root package under src/std/**.
func Files() []*source.File {
	return []*source.File{
		source.NewFile("src/std/prelude/lib.vox", preludeSrc),
		source.NewFile("src/std/testing/lib.vox", testingSrc),
	}
}

const preludeSrc = `// stage0 std/prelude: keep it tiny.
//
// Builtins are intentionally minimal: panic/print.

pub fn assert(cond: bool) -> () {
  if !cond { panic("assertion failed"); }
}

pub fn fail(msg: String) -> () {
  panic(msg);
}

pub fn assert_eq[T](a: T, b: T) -> () {
  if a != b { panic("assertion failed"); }
}
`

const testingSrc = `// stage0 std/testing: implemented in Vox, layered on std/prelude.

import "std/prelude" as prelude

pub fn assert(cond: bool) -> () { prelude.assert(cond); }

pub fn fail(msg: String) -> () { prelude.fail(msg); }

pub fn assert_eq[T](a: T, b: T) -> () { prelude.assert_eq(a, b); }
`

