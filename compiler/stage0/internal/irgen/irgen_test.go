package irgen

import (
	"strings"
	"testing"

	"voxlang/internal/parser"
	"voxlang/internal/source"
	"voxlang/internal/typecheck"
)

func TestLowerWhileBreakContinue(t *testing.T) {
	f := source.NewFile("src/main.vox", `fn main() -> i32 {
  let mut x: i32 = 0;
  while x < 3 {
    x = x + 1;
    if x == 1 { continue; }
    if x == 2 { break; }
  }
  return x;
}`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	irp, err := Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	s := irp.Format()
	for _, sub := range []string{"while_cond_", "while_body_", "while_end_", "condbr"} {
		if !strings.Contains(s, sub) {
			t.Fatalf("expected IR to contain %q; got:\n%s", sub, s)
		}
	}
	// continue and loop back should create multiple branches to while_cond.
	if strings.Count(s, "br while_cond_") < 2 {
		t.Fatalf("expected at least 2 branches to while_cond; got:\n%s", s)
	}
}

func TestLowerMatchArmBlockExpr(t *testing.T) {
	f := source.NewFile("src/main.vox", `enum E { A(i32), None }
fn main() -> i32 {
  let x: E = E.A(1);
  return match x {
    E.A(v) => {
      let y: i32 = v + 1;
      y
    },
    E.None => 0,
  };
}`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	_, err := Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLowerMatchBindPat(t *testing.T) {
	f := source.NewFile("src/main.vox", `enum E { A(i32), None }
fn main() -> i32 {
  let x: E = E.A(1);
  return match x {
    v => match v {
      E.A(n) => n,
      E.None => 0,
    },
  };
}`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	_, err := Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLowerMatchIntAndStrPatterns(t *testing.T) {
	f1 := source.NewFile("src/main.vox", `fn main(x: i32) -> i32 {
	  return match x {
	    -1 => 0,
	    0 => 1,
	    1 => 2,
	    _ => 3,
	  };
	}`)
	prog, pdiags := parser.Parse(f1)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	irp, err := Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(irp.Format(), "-1") {
		t.Fatalf("expected IR to contain -1 const for negative int pattern; got:\n%s", irp.Format())
	}

	f2 := source.NewFile("src/main.vox", `fn main(s: String) -> i32 {
	  return match s {
	    "a" => 1,
	    _ => 0,
	  };
	}`)
	prog, pdiags = parser.Parse(f2)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags = typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	_, err = Generate(checked)
	if err != nil {
		t.Fatal(err)
	}

	f3 := source.NewFile("src/main.vox", `fn main(b: bool) -> i32 {
	  return match b {
	    true => 1,
	    false => 0,
	  };
	}`)
	prog, pdiags = parser.Parse(f3)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags = typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	irp, err = Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(irp.Format(), "cmp_eq") < 1 {
		t.Fatalf("expected bool match to lower via cmp_eq; got:\n%s", irp.Format())
	}
}

func TestLowerMatchNestedVariantPatterns(t *testing.T) {
	f := source.NewFile("src/main.vox", `enum O { Some(i32), None }
enum R { Ok(O), Err(i32) }
fn main(x: R) -> i32 {
  return match x {
    R.Ok(O.Some(v)) => v,
    R.Ok(O.None) => 0,
    R.Err(_) => -1,
  };
}`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	irp, err := Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	s := irp.Format()
	if strings.Count(s, "enum_tag") < 2 {
		t.Fatalf("expected nested enum patterns to use enum_tag checks; got:\n%s", s)
	}
	if strings.Count(s, "enum_payload") < 2 {
		t.Fatalf("expected nested enum patterns to use enum_payload extraction; got:\n%s", s)
	}
	for _, sub := range []string{" Ok ", " Some ", " None ", " Err "} {
		if !strings.Contains(s, sub) {
			t.Fatalf("expected IR to mention variant %q; got:\n%s", sub, s)
		}
	}
}

func TestLowerShortCircuitLogicOps(t *testing.T) {
	f := source.NewFile("src/main.vox", `fn rhs() -> bool { panic("rhs executed"); return true; }
fn main() -> bool {
  // Must lower to condbr + blocks (short-circuit), not eager "and/or" instructions.
  let a: bool = false && rhs();
  let b: bool = true || rhs();
  return a == false && b == true;
}`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := typecheck.Check(prog, typecheck.Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	irp, err := Generate(checked)
	if err != nil {
		t.Fatal(err)
	}
	s := irp.Format()
	if !strings.Contains(s, "condbr") {
		t.Fatalf("expected IR to contain condbr for short-circuit; got:\n%s", s)
	}
	if strings.Contains(s, " = and ") || strings.Contains(s, " = or ") {
		t.Fatalf("expected short-circuit lowering to avoid eager and/or; got:\n%s", s)
	}
}
