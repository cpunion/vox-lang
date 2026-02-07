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
	f := source.NewFile("test.vox", `import "dep"
fn main() -> i32 { return dep.one(); }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
	if len(prog.Imports) != 1 || prog.Imports[0].Path != "dep" {
		t.Fatalf("expected 1 import dep, got %#v", prog.Imports)
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
	mem, ok := call.Callee.(*ast.MemberExpr)
	if !ok {
		t.Fatalf("expected member callee, got %T", call.Callee)
	}
	recv, ok := mem.Recv.(*ast.IdentExpr)
	if !ok {
		t.Fatalf("expected ident recv, got %T", mem.Recv)
	}
	if recv.Name != "dep" || mem.Name != "one" {
		t.Fatalf("unexpected member: %s.%s", recv.Name, mem.Name)
	}
}
