package interp

import (
	"testing"

	"voxlang/internal/parser"
	"voxlang/internal/source"
	"voxlang/internal/typecheck"
)

func TestWhileBreakContinueSemantics(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let mut x: i32 = 0;
  while x < 10 {
    x = x + 1;
    if x == 5 { continue; }
    if x == 9 { break; }
  }
  return x;
}`)
	if out != "9" {
		t.Fatalf("expected 9, got %q", out)
	}
}

func TestNestedLoopsBreakOnlyInner(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let mut x: i32 = 0;
  let mut y: i32 = 0;
  while x < 3 {
    x = x + 1;
    y = 0;
    while y < 10 {
      y = y + 1;
      break;
    }
  }
  return x + y;
}`)
	if out != "4" {
		t.Fatalf("expected 4, got %q", out)
	}
}

func TestContinueSkipsRestOfBody(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let mut x: i32 = 0;
  let mut sum: i32 = 0;
  while x < 5 {
    x = x + 1;
    if x == 3 { continue; }
    sum = sum + x;
  }
  return sum; // 1 + 2 + 4 + 5 = 12
}`)
	if out != "12" {
		t.Fatalf("expected 12, got %q", out)
	}
}

func TestRunTestsIgnoresTestPrefixInNonTestFiles(t *testing.T) {
	f := source.NewFile("src/main.vox", `fn test_not_a_test() -> () { assert(true); }`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	log, err := RunTests(checked)
	if err != nil {
		t.Fatal(err)
	}
	if log != "[test] no tests found\n" {
		t.Fatalf("expected no tests found, got %q", log)
	}
}

func runMain(t *testing.T, src string) string {
	t.Helper()
	f := source.NewFile("src/main.vox", src)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	out, err := RunMain(checked)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
