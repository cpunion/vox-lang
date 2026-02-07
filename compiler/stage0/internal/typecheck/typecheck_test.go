package typecheck

import (
	"testing"

	"voxlang/internal/ast"
	"voxlang/internal/parser"
	"voxlang/internal/source"
)

func TestUntypedIntConstraint(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { let x: i32 = 1; return x; }`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
}

func TestQualifiedCallRequiresImport(t *testing.T) {
	f := source.NewFile("src/main.vox", `fn main() -> i32 { return dep.one(); }`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags == nil || len(tdiags.Items) == 0 {
		t.Fatalf("expected typecheck diagnostics")
	}
	found := false
	for _, it := range tdiags.Items {
		if it.Msg == "unknown package qualifier: dep (did you forget `import \"dep\"`?)" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing import diagnostic, got: %+v", tdiags.Items)
	}
}

func TestImportAliasResolvesCallTarget(t *testing.T) {
	files := []*source.File{
		source.NewFile("src/main.vox", `import "mathlib" as m
fn main() -> i32 { return m.one(); }`),
		source.NewFile("mathlib/src/lib.vox", `fn one() -> i32 { return 1; }`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	var call *ast.CallExpr
	for _, fn := range prog.Funcs {
		if fn.Name != "main" {
			continue
		}
		// return m.one();
		ret := fn.Body.Stmts[0].(*ast.ReturnStmt)
		call = ret.Expr.(*ast.CallExpr)
	}
	if call == nil {
		t.Fatalf("missing call expr")
	}
	if got := checked.CallTargets[call]; got != "mathlib::one" {
		t.Fatalf("expected call target mathlib::one, got %q", got)
	}
}
