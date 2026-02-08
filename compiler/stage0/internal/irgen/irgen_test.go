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
