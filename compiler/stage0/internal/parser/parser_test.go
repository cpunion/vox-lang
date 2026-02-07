package parser

import (
	"testing"

	"voxlang/internal/ast"
	"voxlang/internal/source"
)

func TestParseMain(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
	if prog.Funcs[0].Name != "main" {
		t.Fatalf("expected main, got %q", prog.Funcs[0].Name)
	}
}

func TestParsePathCall(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { return dep::one(); }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
	body := prog.Funcs[0].Body
	if len(body.Stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(body.Stmts))
	}
	ret, ok := body.Stmts[0].(*ast.ReturnStmt)
	if !ok || ret.Expr == nil {
		t.Fatalf("expected return stmt with expr")
	}
	call, ok := ret.Expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected call expr, got %T", ret.Expr)
	}
	path, ok := call.Callee.(*ast.PathExpr)
	if !ok {
		t.Fatalf("expected path callee, got %T", call.Callee)
	}
	if len(path.Parts) != 2 || path.Parts[0] != "dep" || path.Parts[1] != "one" {
		t.Fatalf("unexpected path parts: %#v", path.Parts)
	}
}
