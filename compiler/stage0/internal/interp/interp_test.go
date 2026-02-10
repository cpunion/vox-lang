package interp

import (
	"errors"
	"testing"
	"time"

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

func TestInterpStructCopyDoesNotAliasFieldWrites(t *testing.T) {
	out := runMain(t, `struct S { x: i32 }
fn main() -> i32 {
  let mut a: S = S { x: 1 };
  let mut b: S = a;
  b.x = 2;
  return a.x * 10 + b.x;
}`)
	if out != "12" {
		t.Fatalf("expected 12, got %q", out)
	}
}

func TestInterpNameLookupCacheRespectsShadowing(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let x: i32 = 1;
  {
    let x: i32 = 2;
    assert(x == 2);
  }
  return x;
}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestInterpNameLookupCacheRespectsParamShadowing(t *testing.T) {
	out := runMain(t, `fn f(x: i32) -> i32 {
  return x;
}
fn main() -> i32 {
  let x: i32 = 7;
  return f(3) * 10 + x;
}`)
	if out != "37" {
		t.Fatalf("expected 37, got %q", out)
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

func TestInterpEnumMultiPayload(t *testing.T) {
	out := runMain(t, `enum E { Pair(i32, i32), None }
fn main() -> i32 {
  let x: E = E.Pair(40, 2);
  return match x {
    E.Pair(a, b) => a + b,
    E.None => 0,
  };
}`)
	if out != "42" {
		t.Fatalf("expected 42, got %q", out)
	}
}

func TestInterpMatchIntPatterns(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
	  let x: i32 = 1;
	  return match x { 0 => 10, 1 => 20, _ => 30 };
	}`)
	if out != "20" {
		t.Fatalf("expected 20, got %q", out)
	}
}

func TestInterpMatchI64Patterns(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
		  let x: i64 = 3000000000;
		  return match x { 3000000000 => 1, _ => 0 };
		}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestInterpMatchNegativeIntPattern(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
	  let x: i32 = -1;
	  return match x { -1 => 1, _ => 0 };
	}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestInterpMatchStringPatterns(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
		  let s: String = "a";
		  return match s { "a" => 1, _ => 0 };
		}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestInterpMatchBoolPatterns(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
	  let b: bool = true;
	  return match b {
	    true => 1,
	    false => 0,
	  };
	}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestInterpMatchNestedVariantPatterns(t *testing.T) {
	out := runMain(t, `enum O { Some(i32), None }
enum R { Ok(O), Err(i32) }
fn main() -> i32 {
  let x: R = R.Ok(O.Some(5));
  return match x {
    R.Ok(O.Some(v)) => v,
    R.Ok(O.None) => 0,
    R.Err(_) => -1,
  };
}`)
	if out != "5" {
		t.Fatalf("expected 5, got %q", out)
	}
}

func TestInterpMatchMultipleArmsSameVariant(t *testing.T) {
	out := runMain(t, `enum E { A(i32), None }
fn main() -> i32 {
  let x: E = E.A(0);
  return match x {
    E.A(1) => 1,
    E.A(v) => v,
    E.None => -1,
  };
}`)
	if out != "0" {
		t.Fatalf("expected 0, got %q", out)
	}
}

func TestInterpMatchBindPattern(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
		  let x: i32 = 41;
		  return match x { v => v + 1 };
		}`)
	if out != "42" {
		t.Fatalf("expected 42, got %q", out)
	}
}

func TestInterpStructFieldOfEnumType(t *testing.T) {
	out := runMain(t, `enum K { A, B }
struct S { k: K }
fn main() -> i32 {
  let s: S = S { k: K.A };
  let k: K = s.k;
  return match k { K.A => 1, K.B => 2 };
}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestSummarizeInterpTestResults_ModuleAndSlowest(t *testing.T) {
	testNames := []string{
		"tests::test_fast",
		"tests::test_slow",
		"tests.net::test_fail",
	}
	results := map[string]interpTestResult{
		"tests::test_fast":     {dur: 1 * time.Millisecond, err: nil},
		"tests::test_slow":     {dur: 4 * time.Millisecond, err: nil},
		"tests.net::test_fail": {dur: 2 * time.Millisecond, err: errors.New("boom")},
	}
	mods, slowest := summarizeInterpTestResults(testNames, results)
	if len(mods) != 2 {
		t.Fatalf("module count = %d, want 2", len(mods))
	}
	if mods[0].module != "tests" || mods[0].passed != 2 || mods[0].failed != 0 {
		t.Fatalf("unexpected module summary[0]: %+v", mods[0])
	}
	if mods[1].module != "tests.net" || mods[1].passed != 0 || mods[1].failed != 1 {
		t.Fatalf("unexpected module summary[1]: %+v", mods[1])
	}
	if len(slowest) != 3 {
		t.Fatalf("slowest count = %d, want 3", len(slowest))
	}
	if slowest[0].name != "tests::test_slow" {
		t.Fatalf("slowest[0] = %q, want tests::test_slow", slowest[0].name)
	}
	if slowest[1].name != "tests.net::test_fail" {
		t.Fatalf("slowest[1] = %q, want tests.net::test_fail", slowest[1].name)
	}
	if slowest[2].name != "tests::test_fast" {
		t.Fatalf("slowest[2] = %q, want tests::test_fast", slowest[2].name)
	}
}

func TestInterpModuleWorkers(t *testing.T) {
	if got := interpModuleWorkers(0, 0); got != 1 {
		t.Fatalf("workers(0,0) = %d, want 1", got)
	}
	if got := interpModuleWorkers(3, 2); got != 2 {
		t.Fatalf("workers(3,2) = %d, want 2", got)
	}
	if got := interpModuleWorkers(3, 8); got != 3 {
		t.Fatalf("workers(3,8) = %d, want 3", got)
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

func TestInterpVecCopyDoesNotAliasPush(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let mut a: Vec[i32] = Vec();
  a.push(1);
  let mut b: Vec[i32] = a;
  b.push(2);
  return a.len() * 10 + b.len();
}`)
	if out != "12" {
		t.Fatalf("expected 12, got %q", out)
	}
}

func TestInterpStructFieldVecCopyDoesNotAliasPush(t *testing.T) {
	out := runMain(t, `struct S { items: Vec[i32] }
fn main() -> i32 {
  let mut a: S = S { items: Vec() };
  a.items.push(1);
  let mut b: S = a;
  b.items.push(2);
  return a.items.len() * 10 + b.items.len();
}`)
	if out != "12" {
		t.Fatalf("expected 12, got %q", out)
	}
}

func TestInterpGenericFuncIdInference(t *testing.T) {
	out := runMain(t, `fn id[T](x: T) -> T { return x; }
fn main() -> i32 { return id(41) + 1; }`)
	if out != "42" {
		t.Fatalf("expected 42, got %q", out)
	}
}

func TestInterpStringLenAndByteAt(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let s: String = "abc";
  let n: i32 = s.len();
  let b0: i32 = s.byte_at(0);
  return n + b0; // 3 + 'a'(97) = 100
}`)
	if out != "100" {
		t.Fatalf("expected 100, got %q", out)
	}
}

func TestInterpStringSlice(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let s: String = "abc";
  let t: String = s.slice(1, 3); // "bc"
  return t.len() + t.byte_at(0); // 2 + 'b'(98) = 100
}`)
	if out != "100" {
		t.Fatalf("expected 100, got %q", out)
	}
}

func TestInterpStringOrderedComparison(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let a: String = "a";
  let b: String = "b";
  if a < b && a <= b && b > a && b >= a { return 1; }
  return 0;
}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestShortCircuitAndAndDoesNotEvalRHS(t *testing.T) {
	out := runMain(t, `fn rhs() -> bool { panic("rhs executed"); return true; }
fn main() -> i32 {
  let x: bool = false && rhs();
  return if x { 1 } else { 0 };
}`)
	if out != "0" {
		t.Fatalf("expected 0, got %q", out)
	}
}

func TestShortCircuitOrOrDoesNotEvalRHS(t *testing.T) {
	out := runMain(t, `fn rhs() -> bool { panic("rhs executed"); return true; }
fn main() -> i32 {
  let x: bool = true || rhs();
  return if x { 1 } else { 0 };
}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestBitwiseAndShiftOps(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let a: i32 = 6 & 3;     // 2
  let b: i32 = 1 << 4;    // 16
  let c: i32 = b >> 2;    // 4
  return (a | c) ^ 1;     // (2|4)^1 = 7
}`)
	if out != "7" {
		t.Fatalf("expected 7, got %q", out)
	}
}

func TestCompoundAssignOps(t *testing.T) {
	out := runMain(t, `struct S { x: i32 }
fn main() -> i32 {
  let mut x: i32 = 32;
  x += 10;    // 42
  x -= 2;     // 40
  x *= 3;     // 120
  x /= 5;     // 24
  x %= 7;     // 3
  x <<= 4;    // 48
  x >>= 3;    // 6
  x &= 3;     // 2
  x |= 8;     // 10
  x ^= 15;    // 5
  let mut s: S = S { x: 1 };
  s.x += 2;   // 3
  s.x *= 4;   // 12
  s.x >>= 2;  // 3
  return x + s.x;
}`)
	if out != "8" {
		t.Fatalf("expected 8, got %q", out)
	}
}

func TestConstI8WrappingAdd(t *testing.T) {
	out := runMain(t, `const A: i8 = 120
const B: i8 = A + A
fn main() -> i32 {
  return B as i32;
}`)
	if out != "-16" {
		t.Fatalf("expected -16, got %q", out)
	}
}

func TestConstU64MaxAddWraps(t *testing.T) {
	out := runMain(t, `const A: u64 = 18446744073709551615
const B: u64 = A + 1
fn main() -> i32 {
  if B == 0 { return 1; } else { return 0; }
}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestConstU64ComparisonUsesUnsignedOrder(t *testing.T) {
	out := runMain(t, `const A: u64 = 18446744073709551615
const B: bool = A > 1
fn main() -> i32 {
  if B { return 1; } else { return 0; }
}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestExprCastU64MaxLiteral(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let x: u64 = 18446744073709551615 as u64;
  if x > 1 { return 1; } else { return 0; }
}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestConstCastU64MaxLiteral(t *testing.T) {
	out := runMain(t, `const N: u64 = 18446744073709551615 as u64
fn main() -> i32 {
  if N > 1 { return 1; } else { return 0; }
}`)
	if out != "1" {
		t.Fatalf("expected 1, got %q", out)
	}
}

func TestShiftCountOutOfRangePanics(t *testing.T) {
	_, err := runMainErr(t, `fn main() -> i32 {
  let x: i32 = 1 << 32;
  return x;
}`)
	if err == "" || err != "shift count out of range" {
		t.Fatalf("expected shift count out of range, got %q", err)
	}
}

func TestAsCastI64ToI32Checked(t *testing.T) {
	out := runMain(t, `fn main() -> i32 {
  let x: i64 = 41;
  let y: i32 = x as i32;
  return y + 1;
}`)
	if out != "42" {
		t.Fatalf("expected 42, got %q", out)
	}
}

func TestAsCastI64ToI32OverflowPanics(t *testing.T) {
	_, err := runMainErr(t, `fn main() -> i32 {
  let x: i64 = 3000000000;
  let y: i32 = x as i32;
  return y;
}`)
	if err == "" || err != "i64 to i32 overflow" {
		t.Fatalf("expected overflow error, got %q", err)
	}
}

func TestAsCastIntLiteralOverflowPanics(t *testing.T) {
	_, err := runMainErr(t, `fn main() -> i32 {
  let y: i32 = 3000000000 as i32;
  return y;
}`)
	if err == "" || err != "i64 to i32 overflow" {
		t.Fatalf("expected overflow error, got %q", err)
	}
}

func TestRangeCastI32Ok(t *testing.T) {
	out := runMain(t, `type Tiny = @range(0..=3) i32
fn main() -> i32 {
  let x: i32 = 2;
  let y: Tiny = x as Tiny;
  return y;
}`)
	if out != "2" {
		t.Fatalf("expected 2, got %q", out)
	}
}

func TestRangeCastI32Panics(t *testing.T) {
	_, err := runMainErr(t, `type Tiny = @range(0..=3) i32
fn main() -> i32 {
  let x: i32 = 5;
  let y: Tiny = x as Tiny;
  return y;
}`)
	if err == "" || err != "range check failed" {
		t.Fatalf("expected range check error, got %q", err)
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

func runMainErr(t *testing.T, src string) (string, string) {
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
	out, e := RunMain(checked)
	if e == nil {
		return out, ""
	}
	return out, e.Error()
}
