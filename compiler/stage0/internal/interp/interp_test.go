package interp

import (
	"testing"

	"voxlang/internal/parser"
	"voxlang/internal/source"
	"voxlang/internal/stdlib"
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
	f := source.NewFile("src/main.vox", `fn test_not_a_test() -> () { }`)
	stdFiles, err := stdlib.Files()
	if err != nil {
		t.Fatal(err)
	}
	prog, pdiags := parser.ParseFiles(append(stdFiles, f))
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

func TestInterpStructFieldReadWrite(t *testing.T) {
	out := runMain(t, `struct Point { x: i32, y: i32 }
fn main() -> i32 {
  let mut p: Point = Point { x: 1, y: 2 };
  let a: i32 = p.x;
  p.x = a + 1;
  return p.x + p.y;
}`)
	if out != "4" {
		t.Fatalf("expected 4, got %q", out)
	}
}

func TestInterpEnumCtorAndMatch(t *testing.T) {
	out := runMain(t, `enum E { A(i32), B(String), None }
fn main() -> i32 {
  // enum constructor call + match expression (payload types differ across variants)
  let x: E = E.B("hi");
  let ok: bool = match x {
    E.A(v) => v == 0,
    E.B(s) => s == "hi",
    E.None => false,
  };
  assert(ok);

  let y: E = E.A(41);
  return match y {
    E.A(v) => v + 1,
    E.B(s) => 0,
    E.None => 0,
  };
}`)
	if out != "42" {
		t.Fatalf("expected 42, got %q", out)
	}
}

func TestInterpVecPushLenGet(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let mut v: Vec[i32] = Vec();
  v.push(41);
  v.push(1);
  return v.get(0) + v.get(1) + v.len();
}`)
	// 41 + 1 + 2
	if out != "44" {
		t.Fatalf("expected 44, got %q", out)
	}
}

func TestInterpGenericFuncIdInference(t *testing.T) {
	out := runMain(t, `fn id[T](x: T) -> T { return x; }
fn main() -> i32 { return id(41) + 1; }`)
	if out != "42" {
		t.Fatalf("expected 42, got %q", out)
	}
}

func runMain(t *testing.T, src string) string {
	t.Helper()
	f := source.NewFile("src/main.vox", src)
	stdFiles, err := stdlib.Files()
	if err != nil {
		t.Fatal(err)
	}
	prog, pdiags := parser.ParseFiles(append(stdFiles, f))
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
